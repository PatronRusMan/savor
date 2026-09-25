import { Test, TestingModule } from "@nestjs/testing";
import { CourierService } from "./courier.service";
import { PrismaService } from "../prisma/prisma.service";
import { MqService } from "../mq/mq.service";
import { ForbiddenException } from "@nestjs/common";

describe("CourierService", () => {
  let service: CourierService;
  let prisma: PrismaService;
  let mq: MqService;

  const mockProfile = {
    userId: "courier-1",
    name: "John Doe",
    isOnline: false,
    createdAt: new Date(),
  };

  const mockDelivery = {
    id: "delivery-1",
    orderId: "order-1",
    courierId: "courier-1",
    status: "assigned",
    restaurantName: "Test Restaurant",
    address: "123 Main St",
    assignedAt: new Date(),
    pickedUpAt: null,
    deliveredAt: null,
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        CourierService,
        {
          provide: PrismaService,
          useValue: {
            courierProfile: {
              upsert: jest.fn(),
              update: jest.fn(),
              findMany: jest.fn(),
            },
            delivery: {
              findUnique: jest.fn(),
              findMany: jest.fn(),
              create: jest.fn(),
              update: jest.fn(),
            },
          },
        },
        {
          provide: MqService,
          useValue: {
            consume: jest.fn(),
            publish: jest.fn(),
          },
        },
      ],
    }).compile();

    service = module.get<CourierService>(CourierService);
    prisma = module.get<PrismaService>(PrismaService);
    mq = module.get<MqService>(MqService);
  });

  describe("ensure", () => {
    it("should create or update courier profile", async () => {
      jest.spyOn(prisma.courierProfile, "upsert").mockResolvedValue(mockProfile);

      const result = await service.ensure("courier-1", "John Doe");

      expect(result).toEqual(mockProfile);
      expect(prisma.courierProfile.upsert).toHaveBeenCalledWith({
        where: { userId: "courier-1" },
        update: { name: "John Doe" },
        create: { userId: "courier-1", name: "John Doe", isOnline: false },
      });
    });
  });

  describe("setOnline", () => {
    it("should update courier online status", async () => {
      const onlineProfile = { ...mockProfile, isOnline: true };
      jest.spyOn(prisma.courierProfile, "upsert").mockResolvedValue(mockProfile);
      jest.spyOn(prisma.courierProfile, "update").mockResolvedValue(onlineProfile);

      const result = await service.setOnline("courier-1", "John Doe", true);

      expect(result.isOnline).toBe(true);
      expect(prisma.courierProfile.update).toHaveBeenCalledWith({
        where: { userId: "courier-1" },
        data: { isOnline: true },
      });
    });
  });

  describe("me", () => {
    it("should return courier profile with active jobs", async () => {
      jest.spyOn(prisma.courierProfile, "upsert").mockResolvedValue(mockProfile);
      jest.spyOn(prisma.delivery, "findMany").mockResolvedValue([mockDelivery]);

      const result = await service.me("courier-1", "John Doe");

      expect(result.userId).toBe("courier-1");
      expect(result.activeJobs).toHaveLength(1);
      expect(prisma.delivery.findMany).toHaveBeenCalledWith({
        where: { courierId: "courier-1", status: { in: ["assigned", "picked_up"] } },
        orderBy: { assignedAt: "desc" },
      });
    });
  });

  describe("pickup", () => {
    it("should mark delivery as picked up", async () => {
      const pickedUpDelivery = { ...mockDelivery, status: "picked_up", pickedUpAt: new Date() };
      jest.spyOn(prisma.delivery, "findUnique").mockResolvedValue(mockDelivery);
      jest.spyOn(prisma.delivery, "update").mockResolvedValue(pickedUpDelivery);
      jest.spyOn(mq, "publish").mockResolvedValue(undefined);

      const result = await service.pickup("courier-1", "delivery-1");

      expect(result.status).toBe("picked_up");
      expect(result.pickedUpAt).toBeTruthy();
      expect(mq.publish).toHaveBeenCalledWith("courier.picked_up", expect.objectContaining({
        deliveryId: "delivery-1",
        orderId: "order-1",
        courierId: "courier-1",
        status: "picked_up",
      }));
    });

    it("should throw ForbiddenException when courier mismatch", async () => {
      jest.spyOn(prisma.delivery, "findUnique").mockResolvedValue(mockDelivery);

      await expect(service.pickup("wrong-courier", "delivery-1")).rejects.toThrow(ForbiddenException);
    });

    it("should throw ForbiddenException when status is not assigned", async () => {
      const wrongStatus = { ...mockDelivery, status: "delivered" };
      jest.spyOn(prisma.delivery, "findUnique").mockResolvedValue(wrongStatus);

      await expect(service.pickup("courier-1", "delivery-1")).rejects.toThrow(ForbiddenException);
    });
  });

  describe("deliver", () => {
    it("should mark delivery as delivered", async () => {
      const pickedUpDelivery = { ...mockDelivery, status: "picked_up", pickedUpAt: new Date() };
      const deliveredDelivery = { ...pickedUpDelivery, status: "delivered", deliveredAt: new Date() };
      jest.spyOn(prisma.delivery, "findUnique").mockResolvedValue(pickedUpDelivery);
      jest.spyOn(prisma.delivery, "update").mockResolvedValue(deliveredDelivery);
      jest.spyOn(mq, "publish").mockResolvedValue(undefined);

      const result = await service.deliver("courier-1", "delivery-1");

      expect(result.status).toBe("delivered");
      expect(result.deliveredAt).toBeTruthy();
      expect(mq.publish).toHaveBeenCalledWith("courier.delivered", expect.objectContaining({
        deliveryId: "delivery-1",
        orderId: "order-1",
        status: "delivered",
      }));
    });

    it("should throw ForbiddenException when courier mismatch", async () => {
      jest.spyOn(prisma.delivery, "findUnique").mockResolvedValue(mockDelivery);

      await expect(service.deliver("wrong-courier", "delivery-1")).rejects.toThrow(ForbiddenException);
    });

    it("should throw ForbiddenException when status is not picked_up", async () => {
      jest.spyOn(prisma.delivery, "findUnique").mockResolvedValue(mockDelivery);

      await expect(service.deliver("courier-1", "delivery-1")).rejects.toThrow(ForbiddenException);
    });
  });

  describe("jobs", () => {
    it("should return courier delivery history", async () => {
      const jobs = [mockDelivery, { ...mockDelivery, id: "delivery-2", status: "delivered" }];
      jest.spyOn(prisma.delivery, "findMany").mockResolvedValue(jobs);

      const result = await service.jobs("courier-1");

      expect(result).toHaveLength(2);
      expect(prisma.delivery.findMany).toHaveBeenCalledWith({
        where: { courierId: "courier-1" },
        orderBy: { assignedAt: "desc" },
        take: 50,
      });
    });
  });
});
