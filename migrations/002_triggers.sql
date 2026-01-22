-- Функция для логирования изменений
CREATE OR REPLACE FUNCTION log_item_changes()
RETURNS TRIGGER AS $$
DECLARE
    changes JSONB;
    old_json JSONB;
    new_json JSONB;
BEGIN
    -- Определяем действие
    IF TG_OP = 'INSERT' THEN
        old_json := NULL;
        new_json := to_jsonb(NEW);
        changes := to_jsonb(NEW);
    ELSIF TG_OP = 'UPDATE' THEN
        old_json := to_jsonb(OLD);
        new_json := to_jsonb(NEW);
        
        -- Собираем только измененные поля
        changes := '{}'::JSONB;
        IF OLD.name IS DISTINCT FROM NEW.name THEN
            changes := changes || jsonb_build_object('name', jsonb_build_object('old', OLD.name, 'new', NEW.name));
        END IF;
        IF OLD.description IS DISTINCT FROM NEW.description THEN
            changes := changes || jsonb_build_object('description', jsonb_build_object('old', OLD.description, 'new', NEW.description));
        END IF;
        IF OLD.quantity IS DISTINCT FROM NEW.quantity THEN
            changes := changes || jsonb_build_object('quantity', jsonb_build_object('old', OLD.quantity, 'new', NEW.quantity));
        END IF;
        IF OLD.price IS DISTINCT FROM NEW.price THEN
            changes := changes || jsonb_build_object('price', jsonb_build_object('old', OLD.price, 'new', NEW.price));
        END IF;
        IF OLD.location IS DISTINCT FROM NEW.location THEN
            changes := changes || jsonb_build_object('location', jsonb_build_object('old', OLD.location, 'new', NEW.location));
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        old_json := to_jsonb(OLD);
        new_json := NULL;
        changes := to_jsonb(OLD);
    END IF;
    
    -- Вставляем запись в историю
    INSERT INTO item_history (
        item_id,
        action,
        changed_by,
        old_data,
        new_data,
        changes
    ) VALUES (
        COALESCE(NEW.id, OLD.id),
        TG_OP,
        COALESCE(NEW.created_by, OLD.created_by, CURRENT_USER),
        old_json,
        new_json,
        changes
    );
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Триггер для товаров
DROP TRIGGER IF EXISTS items_change_trigger ON items;
CREATE TRIGGER items_change_trigger
AFTER INSERT OR UPDATE OR DELETE ON items
FOR EACH ROW
EXECUTE FUNCTION log_item_changes();

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер для updated_at
DROP TRIGGER IF EXISTS update_items_updated_at ON items;
CREATE TRIGGER update_items_updated_at
BEFORE UPDATE ON items
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
