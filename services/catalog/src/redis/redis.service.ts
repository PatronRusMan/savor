import { Injectable, OnModuleDestroy } from "@nestjs/common";
import Redis from "ioredis";

@Injectable()
export class RedisService implements OnModuleDestroy {
  readonly client: Redis;

  constructor() {
    this.client = new Redis(process.env.REDIS_URL ?? "redis://redis:6379", { lazyConnect: false, maxRetriesPerRequest: 3 });
  }

  async getJSON<T>(key: string): Promise<T | null> {
    const raw = await this.client.get(key);
    return raw ? (JSON.parse(raw) as T) : null;
  }

  async setJSON(key: string, value: unknown, ttlSec: number) {
    await this.client.set(key, JSON.stringify(value), "EX", ttlSec);
  }

  async delPattern(pattern: string) {
    const keys = await this.client.keys(pattern);
    if (keys.length) await this.client.del(...keys);
  }

  async onModuleDestroy() {
    this.client.disconnect();
  }
}
