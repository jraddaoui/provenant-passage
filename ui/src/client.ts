import { AipsApi, Configuration, ResponseError } from "@/openapi-generator";

export function getPath() {
  const configured = import.meta.env.VITE_API_PATH;
  return configured && !configured.startsWith("$") ? configured : "/api";
}
export const client = new AipsApi(new Configuration({ basePath: getPath() }));

export async function errorMessage(error: unknown): Promise<string> {
  if (error instanceof ResponseError) {
    try {
      const body = await error.response.clone().json();
      if (typeof body.message === "string") return body.message;
    } catch {
      /* Use a status-specific fallback. */
    }
    if (error.response.status === 404)
      return "This AIP or its output could not be found.";
    return `Request failed (${error.response.status}). Please try again.`;
  }
  return "Could not reach the application. Please try again.";
}
