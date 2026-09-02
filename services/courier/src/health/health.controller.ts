import { Controller, Get, Header } from "@nestjs/common";
import client from "prom-client";

const register = new client.Registry();
client.collectDefaultMetrics({ register, prefix: "courier_" });

@Controller()
export class HealthController {
  @Get("/health")
  health() {
    return { status: "ok", service: "courier" };
  }
  @Get("/ready")
  ready() {
    return { status: "ready" };
  }
  @Get("/metrics")
  @Header("Content-Type", "text/plain")
  metrics() {
    return register.metrics();
  }
}
