import { Controller, Get, Headers, Param, Post } from "@nestjs/common";
import { CourierService } from "./courier.service";

@Controller("/courier")
export class CourierController {
  constructor(private readonly svc: CourierService) {}

  @Get("/me")
  me(@Headers("x-user-id") userId: string, @Headers("x-user-name") name: string) {
    return this.svc.me(userId, name || "Courier");
  }

  @Post("/shift/start")
  start(@Headers("x-user-id") userId: string, @Headers("x-user-name") name: string) {
    return this.svc.setOnline(userId, name || "Courier", true);
  }

  @Post("/shift/end")
  end(@Headers("x-user-id") userId: string, @Headers("x-user-name") name: string) {
    return this.svc.setOnline(userId, name || "Courier", false);
  }

  @Get("/jobs")
  jobs(@Headers("x-user-id") userId: string) {
    return this.svc.jobs(userId);
  }

  @Post("/jobs/:id/pickup")
  pickup(@Headers("x-user-id") userId: string, @Param("id") id: string) {
    return this.svc.pickup(userId, id);
  }

  @Post("/jobs/:id/deliver")
  deliver(@Headers("x-user-id") userId: string, @Param("id") id: string) {
    return this.svc.deliver(userId, id);
  }
}
