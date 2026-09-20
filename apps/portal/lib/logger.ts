import { createLogger, format, transports } from "winston";

export const DEBUG = process.env.DEBUG === "true";

const SENSITIVE_KEYS = new Set([
  "password",
  "passwd",
  "pwd",
  "auth",
  "authorization",
  "cookie",
  "set-cookie",
  "x-session-token",
  "credit_card",
  "card_number",
  "cvv",
  "ccv",
  "ssn",
  "private_key",
  "privatekey",
  "phone_number",
  "mobile_number",
  "mobile",
  "phone",
  "tel",
  "telephone",
  "mail",
  "contact_email",
  "user_email",
  "recipient",
  "recipient_email",
]);

const RECIPIENT_LIST_KEYS = new Set(["to", "cc", "bcc"]);

// Catches credential-shaped keys not covered by the exact-match SENSITIVE_KEYS
// set above (e.g. "accessToken", "refresh_token", "client_secret", "x-api-key")
// so a new field doesn't slip through unredacted just because it wasn't
// anticipated verbatim.
const SENSITIVE_KEY_PATTERNS = [
  /token/i,
  /secret/i,
  /api[-_]?key/i,
  /credential/i,
];

const maskSensitiveField = (
  key: string,
  value: unknown,
  seen: WeakSet<object>,
): unknown => {
  const lowerKey = key.toLowerCase();

  if (
    SENSITIVE_KEYS.has(lowerKey) ||
    SENSITIVE_KEY_PATTERNS.some((pattern) => pattern.test(lowerKey))
  ) {
    return "[REDACTED]";
  }

  if (RECIPIENT_LIST_KEYS.has(lowerKey)) {
    if (Array.isArray(value)) {
      return value.map((item) =>
        typeof item === "string" ? "[REDACTED]" : maskSensitiveData(item, seen),
      );
    }
    if (typeof value === "string") {
      return "[REDACTED]";
    }
  }

  return maskSensitiveData(value, seen);
};

export const maskSensitiveData = (
  data: unknown,
  seen: WeakSet<object> = new WeakSet(),
): unknown => {
  if (typeof data !== "object" || data === null) return data;
  if (seen.has(data)) return "[Circular]";
  seen.add(data);
  try {
    if (Array.isArray(data))
      return data.map((item) => maskSensitiveData(item, seen));
    const masked: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(
      data as Record<string, unknown>,
    )) {
      masked[key] = maskSensitiveField(key, value, seen);
    }
    return masked;
  } finally {
    seen.delete(data);
  }
};

const formDataToRecord = (formData: FormData): Record<string, string> => {
  const result: Record<string, string> = {};
  formData.forEach((value, key) => {
    if (
      (typeof File !== "undefined" && value instanceof File) ||
      value instanceof Blob
    ) {
      result[key] = "File";
    } else {
      result[key] = String(value);
    }
  });
  return result;
};

const serializeData = (data: unknown): string => {
  try {
    if (Buffer.isBuffer(data)) return "Buffer";
    if (data instanceof ArrayBuffer) return "ArrayBuffer";
    if (ArrayBuffer.isView(data)) return data.constructor.name;
    if (data instanceof Blob) return "Blob";
    if (typeof FormData !== "undefined" && data instanceof FormData) {
      return JSON.stringify(maskSensitiveData(formDataToRecord(data)));
    }
    if (typeof data === "string") return data;
    if (
      typeof data === "object" &&
      data !== null &&
      typeof (data as any).pipe === "function"
    )
      return "Stream";
    return JSON.stringify(maskSensitiveData(data));
  } catch {
    return "[unserializable]";
  }
};

const serializeFormDataInMeta = format((info) => {
  const reserved = new Set(["level", "message", "timestamp"]);

  for (const [key, value] of Object.entries(info)) {
    if (reserved.has(key)) continue;

    if (typeof FormData !== "undefined" && value instanceof FormData) {
      info[key] = maskSensitiveData(formDataToRecord(value));
    }
  }

  return info;
});

const sanitizeAxiosError = format((info) => {
  const splat = (info as Record<string | symbol, unknown>)[Symbol.for("splat")];
  const errorLike = (info.error ??
    info.err ??
    (Array.isArray(splat) ? splat[0] : undefined)) as
    | Record<string, unknown>
    | undefined;
  if (
    errorLike &&
    typeof errorLike === "object" &&
    "isAxiosError" in errorLike
  ) {
    const { request, response, config, ...safe } = errorLike as Record<
      string,
      unknown
    > & {
      isAxiosError: boolean;
    };
    const sanitized: Record<string, unknown> = { ...safe };
    if (response) {
      sanitized.response = {
        status: (response as Record<string, unknown>).status,
        statusText: (response as Record<string, unknown>).statusText,
        data: maskSensitiveData((response as Record<string, unknown>).data),
      };
    }
    if (config) {
      sanitized.config = {
        url: (config as Record<string, unknown>).url,
        method: (config as Record<string, unknown>).method,
        baseURL: (config as Record<string, unknown>).baseURL,
      };
    }
    // Replace in-place under the original key
    if (info.error) info.error = sanitized;
    else if (info.err) info.err = sanitized;
  }
  return info;
});

const stringifyMetaValues = format((info) => {
  const reserved = new Set(["level", "message", "timestamp"]);

  for (const [key, value] of Object.entries(info)) {
    if (reserved.has(key)) continue;

    if (value !== undefined) {
      info[key] = typeof value === "string" ? value : serializeData(value);
    }
  }

  return info;
});

const logger = createLogger({
  level: process.env.LOG_LEVEL ?? "info",
  // Production intentionally logs nothing today — there's no log drain
  // wired up, so writing JSON to stdout on every API call is pure cost with
  // nobody watching. `silent` short-circuits before formatting/transport
  // writes (winston's Logger#_transform), so this is cheap.
  silent: process.env.NODE_ENV === "production",
  format:
    process.env.NODE_ENV === "production"
      ? format.combine(
          format.timestamp(),
          format.errors({ stack: true }),
          sanitizeAxiosError(),
          stringifyMetaValues(),
          format.json(),
        )
      : format.combine(
          format.colorize(),
          format.timestamp({ format: "HH:mm:ss" }),
          serializeFormDataInMeta(),
          format.printf(({ timestamp, level, message, ...meta }) => {
            const maskedMeta = maskSensitiveData(meta) as Record<
              string,
              unknown
            >;
            return `${timestamp} ${level}: ${message}${
              Object.keys(maskedMeta).length
                ? `\n${JSON.stringify(maskedMeta, null, 2)}`
                : ""
            }`;
          }),
        ),
  transports: [new transports.Console()],
});

export default logger;
