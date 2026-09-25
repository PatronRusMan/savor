# Screenshots Guide

This document describes the screenshots that should be captured for the README.

## How to Capture Screenshots

1. Start the application:
   ```bash
   docker compose up --build
   ```

2. Wait for all services to be healthy (check `docker compose ps`)

3. Open http://localhost in your browser

4. Capture the following screenshots:

### Customer Flow (customer@savor.dev / savor1234)

**File: `docs/customer-browse.png`**
- Navigate to the homepage after login
- Show the restaurant list with multiple restaurants visible
- Should show: restaurant cards with images, ratings, cuisine types

**File: `docs/customer-cart.png`**
- Browse a restaurant menu and add items to cart
- Show the cart with 2-3 items
- Should show: dish names, prices, quantities, total

**File: `docs/customer-order.png`**
- After checkout, show the order page
- Display order status and items
- Should show: order details, status (e.g., "paid", "cooking"), items list

### Restaurant Flow (restaurant@savor.dev / savor1234)

**File: `docs/restaurant-orders.png`**
- Show the restaurant dashboard with pending/active orders
- Should show: order list, order details, action buttons (accept, cooking, ready)

**File: `docs/restaurant-menu.png`**
- Show the menu management view
- Should show: list of dishes with edit/delete buttons, add dish form

### Courier Flow (courier@savor.dev / savor1234)

**File: `docs/courier-dashboard.png`**
- Show courier dashboard with available deliveries
- Should show: online/offline toggle, assigned deliveries list

**File: `docs/courier-delivery.png`**
- Show an active delivery detail
- Should show: restaurant name, customer address, pickup/deliver buttons

## Screenshot Requirements

- Resolution: 1920x1080 or 1440x900 (desktop view)
- Format: PNG
- Show real data (use the demo accounts to create orders)
- Ensure UI is fully loaded before capturing
- Capture after a complete order flow (customer → restaurant → courier)

## Quick Demo Flow for Screenshots

1. Login as customer, order from "Nari" or another restaurant
2. Login as restaurant, accept the order → cooking → ready
3. Login as courier, go online, pick up the order
4. Take screenshots at each step
5. Complete the delivery
