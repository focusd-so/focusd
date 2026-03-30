declare module "@focusd/core" {
  export type WeekdayType = "Sunday";
  export const Timezone: { UTC: "UTC" };
  export const Weekday: { Sunday: "Sunday" };
}

declare module "@focusd/runtime" {
  import { WeekdayType, Timezone, Weekday } from "@focusd/core";
  export { WeekdayType, Timezone, Weekday };

  export interface Runtime {
    time: {
      day(): WeekdayType;
    };
  }
}
