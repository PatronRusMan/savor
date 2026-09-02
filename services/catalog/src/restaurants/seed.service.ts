import { Injectable, Logger, OnModuleInit } from "@nestjs/common";
import { PrismaService } from "../prisma/prisma.service";

const OWNER_NARI = "22222222-2222-2222-2222-222222222222";
const OWNER_EMBER = "44444444-4444-4444-4444-444444444444";

@Injectable()
export class SeedService implements OnModuleInit {
  private readonly log = new Logger(SeedService.name);
  constructor(private readonly prisma: PrismaService) {}

  async onModuleInit() {
    const n = await this.prisma.restaurant.count();
    if (n > 0) return;
    for (const r of restaurants()) {
      const created = await this.prisma.restaurant.create({
        data: {
          id: r.id,
          ownerId: r.ownerId,
          name: r.name,
          slug: r.slug,
          description: r.description,
          cuisine: r.cuisine,
          address: r.address,
          etaMinutes: r.etaMinutes,
          rating: r.rating,
          imageUrl: r.imageUrl,
          dishes: { create: r.dishes },
        },
      });
      this.log.log(`seeded ${created.name}`);
    }
  }
}

function restaurants() {
  return [
    {
      id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
      ownerId: OWNER_NARI,
      name: "Nari",
      slug: "nari",
      description: "Khinkali, mchadi and qvevri wines in a brick courtyard off Plekhanov.",
      cuisine: "Georgian",
      address: "14 Dadiani St, Tbilisi",
      etaMinutes: 32,
      rating: 4.9,
      imageUrl: "https://images.unsplash.com/photo-1555939594-58d7cb561ad1?w=1400",
      dishes: [
        { name: "Kalakuri khinkali", description: "Broth-filled mountain dumplings, 5 pcs", priceCents: 1800, category: "Khinkali" },
        { name: "Mushroom khinkali", description: "Imeretian cheese and forest mushrooms", priceCents: 1600, category: "Khinkali" },
        { name: "Ojakhuri", description: "Pork, potatoes, onions, tkemali", priceCents: 2400, category: "Mains" },
        { name: "Badrijani nigvzit", description: "Walnut-stuffed eggplant", priceCents: 1400, category: "Starters" },
        { name: "Saperavi glass", description: "Kakheti, 150 ml", priceCents: 900, category: "Drinks" },
      ],
    },
    {
      id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
      ownerId: OWNER_NARI,
      name: "Soba Hour",
      slug: "soba-hour",
      description: "Late buckwheat noodles and a six-seat counter.",
      cuisine: "Japanese",
      address: "3 Marjanishvili, Tbilisi",
      etaMinutes: 28,
      rating: 4.7,
      imageUrl: "https://images.unsplash.com/photo-1557872943-16a5ac26437e?w=1400",
      dishes: [
        { name: "Zaru soba", description: "Cold buckwheat, nori, wasabi", priceCents: 1900, category: "Noodles" },
        { name: "Duck tsukemen", description: "Dipping broth, smoked duck", priceCents: 2700, category: "Noodles" },
        { name: "Salmon don", description: "Rice, sesame, spring onion", priceCents: 2500, category: "Rice" },
        { name: "Matcha highball", description: "Hojicha syrup, soda", priceCents: 1100, category: "Drinks" },
      ],
    },
    {
      id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3",
      ownerId: OWNER_NARI,
      name: "Portico",
      slug: "portico",
      description: "Roman pies and bitter greens, oven until 2am.",
      cuisine: "Italian",
      address: "9 Erekle II, Tbilisi",
      etaMinutes: 40,
      rating: 4.6,
      imageUrl: "https://images.unsplash.com/photo-1513104890138-7c749659a591?w=1400",
      dishes: [
        { name: "Cacio e pepe", description: "Tonnarelli, pecorino, black pepper", priceCents: 2200, category: "Pasta" },
        { name: "Diavola", description: "Spicy salami, honey, chili", priceCents: 2300, category: "Pizza" },
        { name: "Burrata", description: "Datterini, basil oil", priceCents: 1800, category: "Starters" },
        { name: "Tiramisu", description: "Marsala, not too sweet", priceCents: 1200, category: "Dessert" },
      ],
    },
    {
      id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4",
      ownerId: OWNER_EMBER,
      name: "Ember Clay Oven",
      slug: "ember",
      description: "Tandoor breads, lamb, and smoked yogurt.",
      cuisine: "Central Asian",
      address: "22 Aghmashenebeli, Tbilisi",
      etaMinutes: 36,
      rating: 4.8,
      imageUrl: "https://images.unsplash.com/photo-1529042410759-befb1204b468?w=1400",
      dishes: [
        { name: "Samarkand plov", description: "Carrot, lamb, chickpeas, raisins", priceCents: 2100, category: "Mains" },
        { name: "Tandoor non", description: "Black sesame, still hot", priceCents: 500, category: "Bread" },
        { name: "Shashlik of lamb", description: "Cumin, charcoal, onion salad", priceCents: 2800, category: "Grill" },
        { name: "Ayran", description: "Salted yogurt, mint", priceCents: 400, category: "Drinks" },
      ],
    },
    {
      id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa5",
      ownerId: OWNER_EMBER,
      name: "Dune",
      slug: "dune",
      description: "Wraps, pickles, and tahini that actually tastes of sesame.",
      cuisine: "Levantine",
      address: "5 Pekini Ave, Tbilisi",
      etaMinutes: 24,
      rating: 4.5,
      imageUrl: "https://images.unsplash.com/photo-1561651823-34feb02250e4?w=1400",
      dishes: [
        { name: "Lamb shawarma", description: "Pickled turnip, amba, laffa", priceCents: 1700, category: "Wraps" },
        { name: "Falafel plate", description: "Four pieces, tahini, salad", priceCents: 1500, category: "Plates" },
        { name: "Hummus with beef", description: "Warm, pine nuts, paprika oil", priceCents: 1600, category: "Starters" },
        { name: "Mint lemonade", description: "No syrup, just fruit", priceCents: 700, category: "Drinks" },
      ],
    },
    {
      id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa6",
      ownerId: OWNER_EMBER,
      name: "Green Room",
      slug: "green-room",
      description: "Vegetable-first cooking, wood grill, no lectures.",
      cuisine: "Vegetarian",
      address: "18 Kikodze St, Tbilisi",
      etaMinutes: 30,
      rating: 4.4,
      imageUrl: "https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=1400",
      dishes: [
        { name: "Charred cabbage", description: "XOish mushroom sauce", priceCents: 1600, category: "Mains" },
        { name: "Beet khinkali", description: "Five, with matsoni", priceCents: 1500, category: "Khinkali" },
        { name: "Herb salad", description: "Tarragon, cucumber, walnuts", priceCents: 1100, category: "Starters" },
        { name: "Kombucha", description: "Quince, house", priceCents: 800, category: "Drinks" },
      ],
    },
  ];
}
