-- Seed data for tenant_sushi_house.
-- Run against tenant_sushi_house. Safe to re-run.
USE tenant_sushi_house;

INSERT INTO menu_items (id, name, description, price_minor)
VALUES
    (1, 'Salmon Nigiri',   '2 pieces', 650),
    (2, 'California Roll', '8 pieces', 780),
    (3, 'Miso Soup',       NULL,       350)
AS new
ON DUPLICATE KEY UPDATE
    name        = new.name,
    description = new.description,
    price_minor = new.price_minor;

INSERT INTO orders (id, order_number, customer_name, order_type, status, total_minor)
VALUES
    (1, 'ORD-0001', 'Seed Customer', 'DINE_IN', 'PREPARING', 1000)
AS new
ON DUPLICATE KEY UPDATE
    order_number  = new.order_number,
    customer_name = new.customer_name,
    order_type    = new.order_type,
    status        = new.status,
    total_minor   = new.total_minor;

INSERT INTO order_items (id, order_id, menu_item_id, item_name, unit_price_minor, quantity, line_total_minor)
VALUES
    (1, 1, 1, 'Salmon Nigiri', 650, 1, 650),
    (2, 1, 3, 'Miso Soup',     350, 1, 350)
AS new
ON DUPLICATE KEY UPDATE
    order_id         = new.order_id,
    menu_item_id     = new.menu_item_id,
    item_name        = new.item_name,
    unit_price_minor = new.unit_price_minor,
    quantity         = new.quantity,
    line_total_minor = new.line_total_minor;
