import * as z from "zod";

import type { Translate } from "@/feature/common/type";

export function createAccountSchema(t: Translate) {
  return z.object({
    name: z.string().trim().min(1, t("nameRequired")),
    description: z.string().trim().min(1, t("descriptionRequired")),
  });
}

export type CreateAccountFormValues = z.infer<
  ReturnType<typeof createAccountSchema>
>;
