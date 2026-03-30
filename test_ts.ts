declare module "@focusd/core" {
  export type WeekdayType = "Sunday";
  export const Timezone: { UTC: "UTC" };
  export const Weekday: { Sunday: "Sunday" };
}

declare module "@focusd/runtime" {
  export { WeekdayType, Timezone, Weekday } from "@focusd/core";
  export const foo = 1;
}
