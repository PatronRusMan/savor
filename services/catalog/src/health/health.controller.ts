import { Controller, Get, Header } from "@nestjs/common";
import client from "prom-client";

const register = new client.Registry();
client.collectDefaultMetrics({ register, prefix: "catalog_" });

@Controller()
export class HealthController {
  @Get("/health")
  health() {
    return { status: "ok", service: "catalog" };
  }

  @Get("/ready")
  ready() {
    return { status: "ready" };
  }

  @Get("/metrics")
  @Header("Content-Type", "text/plain")
  async metrics() {
    return register.metrics();
  }
}
