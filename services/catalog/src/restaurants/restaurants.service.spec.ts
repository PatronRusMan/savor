import { Test, TestingModule } from "@nestjs/testing";
import { RestaurantsService } from "./restaurants.service";
import { PrismaService } from "../prisma/prisma.service";
import { RedisService } from "../redis/redis.service";
import { NotFoundException, ForbiddenException, BadRequestException } from "@nestjs/common";

describe("RestaurantsService", () => {
  let service: RestaurantsService;
  let prisma: PrismaService;
  let redis: RedisService;

  const mockRestaurant = {
    id: "rest-123",
    ownerId: "owner-1",
    name: "Test Kitchen",
    slug: "test-kitchen-xyz",
    description: "Great food",
    cuisine: "Georgian",
    address: "123 Main St",
    etaMinutes: 30,
    rating: 4.5,
    imageUrl: "https://example.com/image.jpg",
    isOpen: true,
    createdAt: new Date(),
  };

  const mockDish = {
    id: "dish-123",
    restaurantId: "rest-123",
    name: "Khachapuri",
    description: "Cheese bread",
    priceCents: 1200,
    category: "Mains",
    isAvailable: true,
    createdAt: new Date(),
    restaurant: mockRestaurant,
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        RestaurantsService,
        {
          provide: PrismaService,
          useValue: {
            restaurant: {
              findMany: jest.fn(),
              findFirst: jest.fn(),
              findUnique: jest.fn(),
              create: jest.fn(),
              update: jest.fn(),
            },
            dish: {
              findMany: jest.fn(),
              findUnique: jest.fn(),
              create: jest.fn(),
              update: jest.fn(),
              delete: jest.fn(),
            },
          },
        },
        {
          provide: RedisService,
          useValue: {
            getJSON: jest.fn(),
            setJSON: jest.fn(),
            delPattern: jest.fn(),
          },
        },
      ],
    }).compile();

    service = module.get<RestaurantsService>(RestaurantsService);
    prisma = module.get<PrismaService>(PrismaService);
    redis = module.get<RedisService>(RedisService);
  });

  describe("list", () => {
    it("should return cached restaurants if available", async () => {
      const cached = [mockRestaurant];
      jest.spyOn(redis, "getJSON").mockResolvedValue(cached);

      const result = await service.list();

      expect(result).toEqual(cached);
      expect(redis.getJSON).toHaveBeenCalledWith("restaurants:list::");
      expect(prisma.restaurant.findMany).not.toHaveBeenCalled();
    });

    it("should fetch and cache restaurants when cache miss", async () => {
      jest.spyOn(redis, "getJSON").mockResolvedValue(null);
      jest.spyOn(prisma.restaurant, "findMany").mockResolvedValue([mockRestaurant]);
      jest.spyOn(redis, "setJSON").mockResolvedValue(undefined);

      const result = await service.list();

      expect(result).toEqual([mockRestaurant]);
      expect(prisma.restaurant.findMany).toHaveBeenCalled();
      expect(redis.setJSON).toHaveBeenCalledWith("restaurants:list::", [mockRestaurant], 30);
    });

    it("should filter by cuisine", async () => {
      jest.spyOn(redis, "getJSON").mockResolvedValue(null);
      jest.spyOn(prisma.restaurant, "findMany").mockResolvedValue([mockRestaurant]);

      await service.list(undefined, "Georgian");

      expect(prisma.restaurant.findMany).toHaveBeenCalledWith({
        where: { cuisine: { equals: "Georgian", mode: "insensitive" } },
        orderBy: { rating: "desc" },
      });
    });

    it("should search by query string", async () => {
      jest.spyOn(redis, "getJSON").mockResolvedValue(null);
      jest.spyOn(prisma.restaurant, "findMany").mockResolvedValue([mockRestaurant]);

      await service.list("khach");

      expect(prisma.restaurant.findMany).toHaveBeenCalledWith({
        where: {
          OR: [
            { name: { contains: "khach", mode: "insensitive" } },
            { description: { contains: "khach", mode: "insensitive" } },
            { cuisine: { contains: "khach", mode: "insensitive" } },
          ],
        },
        orderBy: { rating: "desc" },
      });
    });
  });

  describe("byId", () => {
    it("should return restaurant by id", async () => {
      jest.spyOn(prisma.restaurant, "findFirst").mockResolvedValue(mockRestaurant);

      const result = await service.byId("rest-123");

      expect(result).toEqual(mockRestaurant);
      expect(prisma.restaurant.findFirst).toHaveBeenCalledWith({
        where: { OR: [{ id: "rest-123" }, { slug: "rest-123" }] },
      });
    });

    it("should throw NotFoundException when restaurant not found", async () => {
      jest.spyOn(prisma.restaurant, "findFirst").mockResolvedValue(null);

      await expect(service.byId("nonexistent")).rejects.toThrow(NotFoundException);
    });
  });

  describe("create", () => {
    it("should create a new restaurant with valid data", async () => {
      const createData = {
        name: "New Place",
        description: "Great food",
        cuisine: "Italian",
        address: "456 Oak St",
        etaMinutes: 40,
      };

      jest.spyOn(prisma.restaurant, "create").mockResolvedValue({
        ...mockRestaurant,
        ...createData,
        id: "new-rest",
        ownerId: "owner-2",
        slug: "new-place-abcd",
      });
      jest.spyOn(redis, "delPattern").mockResolvedValue(undefined);

      const result = await service.create("owner-2", createData);

      expect(result.name).toBe(createData.name);
      expect(result.ownerId).toBe("owner-2");
      expect(prisma.restaurant.create).toHaveBeenCalled();
      expect(redis.delPattern).toHaveBeenCalledWith("restaurants:list:*");
    });

    it("should throw BadRequestException when name is missing", async () => {
      await expect(
        service.create("owner-1", { name: "", cuisine: "Italian", description: "", address: "" })
      ).rejects.toThrow(BadRequestException);
    });

    it("should throw BadRequestException when cuisine is missing", async () => {
      await expect(
        service.create("owner-1", { name: "Test", cuisine: "", description: "", address: "" })
      ).rejects.toThrow(BadRequestException);
    });
  });

  describe("update", () => {
    it("should update restaurant when owner matches", async () => {
      const updates = { name: "Updated Name", isOpen: false };
      jest.spyOn(prisma.restaurant, "findUnique").mockResolvedValue(mockRestaurant);
      jest.spyOn(prisma.restaurant, "update").mockResolvedValue({ ...mockRestaurant, ...updates });
      jest.spyOn(redis, "delPattern").mockResolvedValue(undefined);

      const result = await service.update("rest-123", "owner-1", "restaurant", updates);

      expect(result.name).toBe("Updated Name");
      expect(result.isOpen).toBe(false);
      expect(redis.delPattern).toHaveBeenCalledWith("restaurants:list:*");
    });

    it("should throw ForbiddenException when owner mismatch", async () => {
      jest.spyOn(prisma.restaurant, "findUnique").mockResolvedValue(mockRestaurant);

      await expect(service.update("rest-123", "wrong-owner", "restaurant", {})).rejects.toThrow(
        ForbiddenException
      );
    });

    it("should throw NotFoundException when restaurant not found", async () => {
      jest.spyOn(prisma.restaurant, "findUnique").mockResolvedValue(null);

      await expect(service.update("nonexistent", "owner-1", "restaurant", {})).rejects.toThrow(
        NotFoundException
      );
    });
  });

  describe("addDish", () => {
    it("should add dish when owner matches", async () => {
      const dishData = { name: "Khinkali", priceCents: 800, description: "Dumplings", category: "Starters" };
      jest.spyOn(prisma.restaurant, "findUnique").mockResolvedValue(mockRestaurant);
      jest.spyOn(prisma.dish, "create").mockResolvedValue({ ...mockDish, ...dishData });

      const result = await service.addDish("rest-123", "owner-1", dishData);

      expect(result.name).toBe(dishData.name);
      expect(prisma.dish.create).toHaveBeenCalledWith({
        data: {
          restaurantId: "rest-123",
          name: dishData.name,
          description: dishData.description,
          priceCents: dishData.priceCents,
          category: dishData.category,
        },
      });
    });

    it("should throw ForbiddenException when owner mismatch", async () => {
      jest.spyOn(prisma.restaurant, "findUnique").mockResolvedValue(mockRestaurant);

      await expect(
        service.addDish("rest-123", "wrong-owner", { name: "Test", priceCents: 1000 })
      ).rejects.toThrow(ForbiddenException);
    });

    it("should throw BadRequestException when name is missing", async () => {
      jest.spyOn(prisma.restaurant, "findUnique").mockResolvedValue(mockRestaurant);

      await expect(
        service.addDish("rest-123", "owner-1", { name: "", priceCents: 1000 })
      ).rejects.toThrow(BadRequestException);
    });
  });

  describe("patchDish", () => {
    it("should update dish when owner matches", async () => {
      const updates = { priceCents: 1500, isAvailable: false };
      jest.spyOn(prisma.dish, "findUnique").mockResolvedValue(mockDish);
      jest.spyOn(prisma.dish, "update").mockResolvedValue({ ...mockDish, ...updates });

      const result = await service.patchDish("dish-123", "owner-1", updates);

      expect(result.priceCents).toBe(1500);
      expect(result.isAvailable).toBe(false);
    });

    it("should throw ForbiddenException when owner mismatch", async () => {
      jest.spyOn(prisma.dish, "findUnique").mockResolvedValue(mockDish);

      await expect(service.patchDish("dish-123", "wrong-owner", {})).rejects.toThrow(ForbiddenException);
    });

    it("should throw NotFoundException when dish not found", async () => {
      jest.spyOn(prisma.dish, "findUnique").mockResolvedValue(null);

      await expect(service.patchDish("nonexistent", "owner-1", {})).rejects.toThrow(NotFoundException);
    });
  });

  describe("deleteDish", () => {
    it("should delete dish when owner matches", async () => {
      jest.spyOn(prisma.dish, "findUnique").mockResolvedValue(mockDish);
      jest.spyOn(prisma.dish, "delete").mockResolvedValue(mockDish);

      const result = await service.deleteDish("dish-123", "owner-1");

      expect(result).toEqual({ status: "ok" });
      expect(prisma.dish.delete).toHaveBeenCalledWith({ where: { id: "dish-123" } });
    });

    it("should throw ForbiddenException when owner mismatch", async () => {
      jest.spyOn(prisma.dish, "findUnique").mockResolvedValue(mockDish);

      await expect(service.deleteDish("dish-123", "wrong-owner")).rejects.toThrow(ForbiddenException);
    });
  });

  describe("menu", () => {
    it("should return restaurant with dishes", async () => {
      jest.spyOn(prisma.restaurant, "findFirst").mockResolvedValue(mockRestaurant);
      jest.spyOn(prisma.dish, "findMany").mockResolvedValue([mockDish]);

      const result = await service.menu("rest-123");

      expect(result.restaurant).toEqual(mockRestaurant);
      expect(result.dishes).toHaveLength(1);
      expect(result.dishes[0].name).toBe("Khachapuri");
    });
  });
});
