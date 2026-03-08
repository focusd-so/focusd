import { IconSparkles } from "@tabler/icons-react";

export function GeneralSettings() {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <IconSparkles className="w-10 h-10 text-muted-foreground/60 mb-3" />
      <p className="font-medium text-muted-foreground">Coming soon</p>
      <p className="text-sm text-muted-foreground/70 mt-1">
        General preferences will be available in a future update.
      </p>
    </div>
  );
}
