import { ReadConfigFileSync } from "../../../bindings/github.com/focusd-so/focusd/internal/fs/service";

export const RUNTIME_TYPES_FILE_PATH = "file:///focusd-runtime.d.ts";

export async function fetchRuntimeTypes(): Promise<string> {
  try {
    return await ReadConfigFileSync("types.d.ts");
  } catch (error) {
    console.error("Failed to load runtime types", error);
    return "";
  }
}
