CREATE TABLE IF NOT EXISTS carts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    restaurant_id UUID,
    restaurant_name TEXT,
    restaurant_owner_id UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cart_items (
    id UUID PRIMARY KEY,
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    dish_id UUID NOT NULL,
    name TEXT NOT NULL,
    price_cents INT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    UNIQUE (cart_id, dish_id)
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    customer_name TEXT NOT NULL,
    restaurant_id UUID NOT NULL,
    restaurant_name TEXT NOT NULL,
    restaurant_owner_id UUID NOT NULL,
    courier_id UUID,
    courier_name TEXT,
    status TEXT NOT NULL,
    total_cents INT NOT NULL,
    address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    dish_id UUID NOT NULL,
    name TEXT NOT NULL,
    price_cents INT NOT NULL,
    quantity INT NOT NULL
);

CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS orders_customer_idx ON orders (customer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS orders_restaurant_owner_idx ON orders (restaurant_owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS orders_courier_idx ON orders (courier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS outbox_unpublished_idx ON outbox (created_at) WHERE published_at IS NULL;
