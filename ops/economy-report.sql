-- Read-only operational economy report. Run with psql -X -v ON_ERROR_STOP=1 -f.
-- Credits/debits are actual wallet flows, including transfers; they are not
-- necessarily gold minted/destroyed. Player IDs are pseudonyms, never classified
-- as people or service identities from names or behavior.
BEGIN READ ONLY;
SET LOCAL statement_timeout='60s';
SET LOCAL TIME ZONE 'UTC';

SELECT 'ledger_coverage' AS report,economy_epoch,revision,source,currency,
       count(*) AS changes,count(DISTINCT transaction_id) AS transactions,
       min(created_at) AS first_record,max(created_at) AS last_record,
       count(*) FILTER (WHERE request_id='') AS changes_without_request_id
FROM public.economy_ledger
GROUP BY economy_epoch,revision,source,currency ORDER BY economy_epoch,revision,source,currency;

SELECT 'player_gold_flows' AS report,economy_epoch,substr(md5(client_uid),1,12) AS player,
       sum(GREATEST(delta,0)) AS credits,sum(GREATEST(-delta,0)) AS debits,sum(delta) AS net,
       count(DISTINCT transaction_id) AS transactions,
       count(DISTINCT created_at::date) AS utc_days_with_gold_changes,
       NULL::numeric AS gold_per_player_active_hour,
       'Missing player-active-hours denominator; timestamps and simulation runtime are not engagement' AS rate_note
FROM public.economy_ledger
WHERE currency='gold' AND source<>'economy_reset'
GROUP BY economy_epoch,client_uid ORDER BY economy_epoch,net DESC;

SELECT 'gold_flows_by_source' AS report,economy_epoch,revision,source,
       sum(GREATEST(delta,0)) AS credits,sum(GREATEST(-delta,0)) AS debits,sum(delta) AS net,
       count(DISTINCT client_uid) AS accounts,count(DISTINCT transaction_id) AS transactions
FROM public.economy_ledger WHERE currency='gold' AND source<>'economy_reset'
GROUP BY economy_epoch,revision,source ORDER BY economy_epoch,credits DESC;

SELECT 'terminal_runs_by_cohort' AS report,
       COALESCE(audit_data#>>'{cohort,economy_epoch}','unknown') AS economy_epoch,
       COALESCE(audit_data#>>'{cohort,build_revision}','unknown') AS revision,
       tier,(depth/10)*10 AS depth_band_start,count(*) AS terminal_runs,
       count(*) FILTER(WHERE victory) AS banked_runs,
       count(*) FILTER(WHERE NOT victory AND end_reason IN ('defeat','revive_failed','timeout','conceded')) AS known_deaths,
       count(*) FILTER(WHERE NOT victory AND end_reason NOT IN ('defeat','revive_failed','timeout','conceded')) AS other_unbanked_endings,
       round(100.0*count(*) FILTER(WHERE NOT victory AND end_reason IN ('defeat','revive_failed','timeout','conceded'))/NULLIF(count(*),0),2) AS known_death_percent_of_terminal_runs,
       sum(COALESCE((audit_data#>>'{timing,resolution_ns}')::numeric,0))/1000000000 AS measured_resolution_seconds,
       sum(COALESCE((audit_data#>>'{timing,estimated_action_window_ns}')::numeric,0))/1000000000 AS estimated_countdown_seconds,
       sum(COALESCE((audit_data#>>'{timing,measured_floors}')::integer,0)) AS measured_floor_records,
       count(*) FILTER(WHERE audit_data#>>'{timing,measured_floors}' IS NULL) AS runs_without_timing_schema,
       'Terminal-run denominator; not deaths per attempted floor. Resolution is server wall runtime including I/O; countdown is only estimated engagement.' AS denominator_note
FROM public.abyss_runs GROUP BY 2,3,tier,depth_band_start ORDER BY 2,3,tier,depth_band_start;

WITH archives AS (
 SELECT value::jsonb AS a FROM public.app_meta
 WHERE left(key,length('abyss_live_replay_session_'))='abyss_live_replay_session_'
)
SELECT 'replay_outcomes' AS report,
       COALESCE(a#>>'{state,cohort,economy_epoch}','unknown') AS economy_epoch,
       COALESCE(a#>>'{state,cohort,build_revision}','unknown') AS revision,
       a#>>'{state,snapshot,phase}' AS phase,
       a#>>'{state,snapshot,result,victory}' AS victory,
       count(*) AS archived_sessions,
       'Archive coverage only; failures without archives cannot be inferred' AS coverage_note
FROM archives GROUP BY 2,3,4,5 ORDER BY 2,3,4,5;

SELECT 'inventory_coverage' AS report,substr(md5(u.client_uid),1,12) AS player,
       (SELECT count(*) FROM public.user_inventory i WHERE i.client_uid=u.client_uid) AS backpack_items,
       (SELECT count(*) FROM public.user_gear g WHERE g.client_uid=u.client_uid) AS equipped_items,
       NULL::integer AS meaningful_upgrades,
       'Requires canonical gear/build/set/passive comparison; raw CR, rarity and item count are not upgrade denominators' AS upgrade_note
FROM public.users u ORDER BY player;
COMMIT;
