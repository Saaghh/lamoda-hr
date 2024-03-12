CREATE TABLE warehouses(
    id uuid primary key,
    name varchar not null,
    is_active bool not null,
    created_at timestamp with time zone not null default now()
);

CREATE TABLE products (
    sku varchar (12) not null primary key,
    name varchar (255) not null,
    size varchar,
    created_at timestamp with time zone not null default now()
);

CREATE TABLE stocks (
    primary key (warehouse_id, product_id),
    warehouse_id uuid not null references warehouses(id),
    product_id varchar (12) references products(sku),
    quantity int not null check ( quantity >= 0 ),
    reserved_quantity int not null check ( reserved_quantity <= stocks.quantity ) check ( reserved_quantity >= 0 ),
    created_at timestamp with time zone not null default now(),
    modified_at timestamp with time zone not null default now()
);

CREATE TABLE reservations (
    id uuid primary key,
    warehouse_id uuid not null references warehouses(id),
    product_id varchar (12) references products(sku),
    quantity int not null check ( quantity > 0 ),
    is_active bool not null default true,
    created_at timestamp with time zone not null default now(),
    due_date timestamp with time zone not null
);

CREATE INDEX idx_reservations_is_active_due_date ON reservations (is_active, due_date);

CREATE TABLE gorp_migrations (
    id varchar primary key,
    applied_at timestamp with time zone
);
