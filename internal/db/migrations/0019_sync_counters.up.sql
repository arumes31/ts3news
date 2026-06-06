-- Triggers to keep denormalized counters in sync automatically

-- Trigger function for unique items counter
CREATE OR REPLACE FUNCTION sync_unique_items_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE users SET unique_items_count = unique_items_count + 1 
        WHERE client_uid = NEW.client_uid;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE users SET unique_items_count = unique_items_count - 1 
        WHERE client_uid = OLD.client_uid;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger for unique items counter
DROP TRIGGER IF EXISTS unique_items_count_trigger ON user_unique_items;
CREATE TRIGGER unique_items_count_trigger
AFTER INSERT OR DELETE ON user_unique_items
FOR EACH ROW EXECUTE FUNCTION sync_unique_items_count();

-- Trigger function for ultimate skills counter
CREATE OR REPLACE FUNCTION sync_ultimate_skills_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE users SET ultimate_skills_count = ultimate_skills_count + 1 
        WHERE client_uid = NEW.client_uid;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE users SET ultimate_skills_count = ultimate_skills_count - 1 
        WHERE client_uid = OLD.client_uid;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger for ultimate skills counter
DROP TRIGGER IF EXISTS ultimate_skills_count_trigger ON user_ultimate_skills;
CREATE TRIGGER ultimate_skills_count_trigger
AFTER INSERT OR DELETE ON user_ultimate_skills
FOR EACH ROW EXECUTE FUNCTION sync_ultimate_skills_count();