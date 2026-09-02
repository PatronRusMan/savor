# Domain events on topic exchange `savor.events`

## order.created
Published by order (outbox) after checkout.
```json
{
  "orderId": "uuid",
  "customerId": "uuid",
  "customerName": "Nia Beridze",
  "restaurantId": "uuid",
  "restaurantName": "Nari",
  "restaurantOwnerId": "uuid",
  "totalCents": 4200,
  "address": "12 Rustaveli Ave, Tbilisi",
  "status": "pending_payment"
}
```

## payment.succeeded / payment.failed
Published by payment after charging (idempotent on orderId).

## order.paid / order.payment_failed / order.accepted / order.cooking / order.ready / order.cancelled / order.assigned / order.picked_up / order.delivered
Published by order outbox after a state transition.

## courier.assigned / courier.picked_up / courier.delivered
Published by courier. Order consumes these to move the ticket.
