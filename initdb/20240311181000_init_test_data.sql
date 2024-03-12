INSERT INTO warehouses (id, name, is_active, created_at) VALUES
    ('974ad127-6b63-48e6-abc1-bdca34ae4435', 'Warehouse 1', true, now()),
    ('7bd7a200-9843-4155-9181-2152ab774ff7', 'Warehouse 2', true, now()),
    ('0c51eb03-4759-4756-ad4c-927e05f4c329', 'Warehouse 3', true, now());

INSERT INTO products (sku, name, size, created_at) VALUES
    ('SKU123456', 'Product 1', 'Size 1', now()),
    ('SKU123457', 'Product 2', 'Size 2', now()),
    ('SKU123458', 'Product 3', 'Size 3', now());

INSERT INTO stocks (warehouse_id, product_id, quantity, reserved_quantity, created_at, modified_at) VALUES
    ('974ad127-6b63-48e6-abc1-bdca34ae4435', 'SKU123456', 100, 0, now(), now()),
    ('974ad127-6b63-48e6-abc1-bdca34ae4435', 'SKU123457', 100, 0, now(), now()),
    ('974ad127-6b63-48e6-abc1-bdca34ae4435', 'SKU123458', 100, 0, now(), now()),

    ('7bd7a200-9843-4155-9181-2152ab774ff7', 'SKU123456', 200, 0, now(), now()),
    ('7bd7a200-9843-4155-9181-2152ab774ff7', 'SKU123457', 200, 0, now(), now()),
    ('7bd7a200-9843-4155-9181-2152ab774ff7', 'SKU123458', 200, 0, now(), now()),

    ('0c51eb03-4759-4756-ad4c-927e05f4c329', 'SKU123456', 300, 0, now(), now()),
    ('0c51eb03-4759-4756-ad4c-927e05f4c329', 'SKU123457', 300, 0, now(), now()),
    ('0c51eb03-4759-4756-ad4c-927e05f4c329', 'SKU123458', 300, 0, now(), now());

INSERT INTO gorp_migrations (id, applied_at) VALUES
    ('20240307133600_init.sql', now())