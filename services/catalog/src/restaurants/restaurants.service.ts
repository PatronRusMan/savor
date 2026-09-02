import { Injectable, NotFoundException, ForbiddenException, BadRequestException } from "@nestjs/common";
import { PrismaService } from "../prisma/prisma.service";
import { RedisService } from "../redis/redis.service";

export type RestaurantDTO = {
  id: string;
  ownerId: string;
  name: string;
  slug: string;
  description: string;
  cuisine: string;
  address: string;
  etaMinutes: number;
  rating: number;
  imageUrl: string;
  isOpen: boolean;
};

@Injectable()
export class RestaurantsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly redis: RedisService,
  ) {}

  toDTO(r: {
    id: string; ownerId: string; name: string; slug: string; description: string;
    cuisine: string; address: string; etaMinutes: number; rating: number; imageUrl: string; isOpen: boolean;
  }): RestaurantDTO {
    return {
      id: r.id, ownerId: r.ownerId, name: r.name, slug: r.slug, description: r.description,
      cuisine: r.cuisine, address: r.address, etaMinutes: r.etaMinutes, rating: r.rating,
      imageUrl: r.imageUrl, isOpen: r.isOpen,
    };
  }

  async list(q?: string, cuisine?: string) {
    const key = `restaurants:list:${q ?? ""}:${cuisine ?? ""}`;
    const cached = await this.redis.getJSON<RestaurantDTO[]>(key);
    if (cached) return cached;
    const rows = await this.prisma.restaurant.findMany({
      where: {
        ...(cuisine ? { cuisine: { equals: cuisine, mode: "insensitive" } } : {}),
        ...(q
          ? {
              OR: [
                { name: { contains: q, mode: "insensitive" } },
                { description: { contains: q, mode: "insensitive" } },
                { cuisine: { contains: q, mode: "insensitive" } },
              ],
            }
          : {}),
      },
      orderBy: { rating: "desc" },
    });
    const dto = rows.map((r) => this.toDTO(r));
    await this.redis.setJSON(key, dto, 30);
    return dto;
  }

  async byId(id: string) {
    const r = await this.prisma.restaurant.findFirst({
      where: { OR: [{ id }, { slug: id }] },
    });
    if (!r) throw new NotFoundException({ error: { code: "not_found", message: "restaurant not found" } });
    return this.toDTO(r);
  }

  async menu(id: string) {
    const r = await this.byId(id);
    const dishes = await this.prisma.dish.findMany({
      where: { restaurantId: r.id },
      orderBy: [{ category: "asc" }, { name: "asc" }],
    });
    return {
      restaurant: r,
      dishes: dishes.map((d) => ({
        id: d.id,
        restaurantId: d.restaurantId,
        name: d.name,
        description: d.description,
        priceCents: d.priceCents,
        category: d.category,
        isAvailable: d.isAvailable,
      })),
    };
  }

  async mine(ownerId: string) {
    const rows = await this.prisma.restaurant.findMany({ where: { ownerId }, orderBy: { name: "asc" } });
    return rows.map((r) => this.toDTO(r));
  }

  async create(ownerId: string, body: { name: string; description: string; cuisine: string; address: string; etaMinutes?: number; imageUrl?: string }) {
    if (!body.name || !body.cuisine) throw new BadRequestException({ error: { code: "invalid", message: "name and cuisine required" } });
    const slug = slugify(body.name) + "-" + Math.random().toString(36).slice(2, 6);
    const r = await this.prisma.restaurant.create({
      data: {
        ownerId,
        name: body.name,
        slug,
        description: body.description ?? "",
        cuisine: body.cuisine,
        address: body.address ?? "",
        etaMinutes: body.etaMinutes ?? 35,
        imageUrl: body.imageUrl ?? "https://images.unsplash.com/photo-1504674900247-0877df9cc836?w=1200",
      },
    });
    await this.redis.delPattern("restaurants:list:*");
    return this.toDTO(r);
  }

  async update(id: string, ownerId: string, role: string, body: Record<string, unknown>) {
    const r = await this.prisma.restaurant.findUnique({ where: { id } });
    if (!r) throw new NotFoundException({ error: { code: "not_found", message: "restaurant not found" } });
    if (role !== "restaurant" || r.ownerId !== ownerId) {
      throw new ForbiddenException({ error: { code: "forbidden", message: "not your restaurant" } });
    }
    const updated = await this.prisma.restaurant.update({
      where: { id },
      data: {
        name: (body.name as string) ?? r.name,
        description: (body.description as string) ?? r.description,
        cuisine: (body.cuisine as string) ?? r.cuisine,
        address: (body.address as string) ?? r.address,
        isOpen: typeof body.isOpen === "boolean" ? body.isOpen : r.isOpen,
        etaMinutes: typeof body.etaMinutes === "number" ? body.etaMinutes : r.etaMinutes,
      },
    });
    await this.redis.delPattern("restaurants:list:*");
    return this.toDTO(updated);
  }

  async addDish(restaurantId: string, ownerId: string, body: { name: string; description?: string; priceCents: number; category?: string }) {
    const r = await this.prisma.restaurant.findUnique({ where: { id: restaurantId } });
    if (!r || r.ownerId !== ownerId) throw new ForbiddenException({ error: { code: "forbidden", message: "not your restaurant" } });
    if (!body.name || !body.priceCents) throw new BadRequestException({ error: { code: "invalid", message: "name and price required" } });
    return this.prisma.dish.create({
      data: {
        restaurantId,
        name: body.name,
        description: body.description ?? "",
        priceCents: body.priceCents,
        category: body.category ?? "Mains",
      },
    });
  }

  async patchDish(id: string, ownerId: string, body: Record<string, unknown>) {
    const d = await this.prisma.dish.findUnique({ where: { id }, include: { restaurant: true } });
    if (!d) throw new NotFoundException({ error: { code: "not_found", message: "dish not found" } });
    if (d.restaurant.ownerId !== ownerId) throw new ForbiddenException({ error: { code: "forbidden", message: "not your dish" } });
    return this.prisma.dish.update({
      where: { id },
      data: {
        name: (body.name as string) ?? d.name,
        description: (body.description as string) ?? d.description,
        priceCents: typeof body.priceCents === "number" ? body.priceCents : d.priceCents,
        category: (body.category as string) ?? d.category,
        isAvailable: typeof body.isAvailable === "boolean" ? body.isAvailable : d.isAvailable,
      },
    });
  }

  async deleteDish(id: string, ownerId: string) {
    const d = await this.prisma.dish.findUnique({ where: { id }, include: { restaurant: true } });
    if (!d) throw new NotFoundException({ error: { code: "not_found", message: "dish not found" } });
    if (d.restaurant.ownerId !== ownerId) throw new ForbiddenException({ error: { code: "forbidden", message: "not your dish" } });
    await this.prisma.dish.delete({ where: { id } });
    return { status: "ok" };
  }

  async dish(id: string) {
    const d = await this.prisma.dish.findUnique({ where: { id } });
    if (!d) throw new NotFoundException({ error: { code: "not_found", message: "dish not found" } });
    return {
      id: d.id,
      restaurantId: d.restaurantId,
      name: d.name,
      description: d.description,
      priceCents: d.priceCents,
      category: d.category,
      isAvailable: d.isAvailable,
    };
  }
}

function slugify(s: string) {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");
}
