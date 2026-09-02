import { Injectable, Logger, OnModuleInit } from "@nestjs/common";
import nodemailer, { Transporter } from "nodemailer";

@Injectable()
export class MailService implements OnModuleInit {
  private readonly log = new Logger(MailService.name);
  private transporter!: Transporter;
  private from = process.env.SMTP_FROM ?? "Savor <noreply@savor.dev>";

  onModuleInit() {
    this.transporter = nodemailer.createTransport({
      host: process.env.SMTP_HOST ?? "mailhog",
      port: Number(process.env.SMTP_PORT ?? 1025),
      secure: false,
    });
  }

  async send(to: string, subject: string, text: string) {
    try {
      await this.transporter.sendMail({ from: this.from, to, subject, text });
    } catch (err) {
      this.log.warn(`mail failed: ${(err as Error).message}`);
    }
  }
}
