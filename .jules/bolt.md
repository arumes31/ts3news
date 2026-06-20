## 2024-05-15 - Removed redundant calculateTotalStats calls per user
**Learning:** In headless TS3 bot architectures processing all connected clients per cycle, doing identical expensive database reads (calculating total stats across gears, effects, and levels) multiple times in a single logical loop represents a huge overhead when N users are processed.
**Action:** Always verify if data gathered during the initial compilation of objects (e.g., `UserInCombat`) can be cached within those objects to prevent executing identical multi-join database lookups later in the same transaction loop.
