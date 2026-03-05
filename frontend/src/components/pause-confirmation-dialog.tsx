import { useState, useMemo, useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  IconAlertTriangle,
  IconClock,
  IconShieldOff,
  IconBulb,
} from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogBody,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { useUsageStore } from "@/stores/usage-store";

interface PauseConfirmationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PAUSE_OPTIONS = [
  { label: "15 min", value: 15 },
  { label: "30 min", value: 30 },
  { label: "1 hour", value: 60 },
];

export function PauseConfirmationDialog({
  open,
  onOpenChange,
}: PauseConfirmationDialogProps) {
  const getBlockedItemsList = useUsageStore((state) => state.getBlockedItemsList);
  const addToWhitelist = useUsageStore((state) => state.addToWhitelist);
  const pauseProtection = useUsageStore((state) => state.pauseProtection);
  const pauseHistory = useUsageStore((state) => state.pauseHistory);
  const getPauseHistory = useUsageStore((state) => state.getPauseHistory);
  const navigate = useNavigate();
  const [selectedDuration, setSelectedDuration] = useState<number | null>(null);
  const [isPausing, setIsPausing] = useState(false);
  const [allowingKey, setAllowingKey] = useState<string | null>(null);

  // Fetch pause history for the last 30 days when the dialog opens
  // and reset selected duration when the dialog is closed.
  useEffect(() => {
    if (open) {
      getPauseHistory(30);
    } else {
      setSelectedDuration(null);
    }
  }, [open, getPauseHistory]);

  // Get last 2 blocked items to show as quick allow options
  const blockedItems = getBlockedItemsList().slice(0, 2);

  const { todayCount, weekCount } = useMemo(() => {
    if (!pauseHistory) return { todayCount: 0, weekCount: 0 };

    let tCount = 0;
    let wCount = 0;

    // Create Date objects for midnight today and 7 days ago
    const now = new Date();
    const startOfTodaySeconds = Math.floor(new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime() / 1000);
    const oneWeekAgoSeconds = Math.floor(Date.now() / 1000) - 7 * 24 * 60 * 60;

    pauseHistory.forEach((p) => {
      if (p.created_at >= startOfTodaySeconds) {
        tCount++;
      }
      if (p.created_at >= oneWeekAgoSeconds) {
        wCount++;
      }
    });

    return { todayCount: tCount, weekCount: wCount };
  }, [pauseHistory]);

  const handleConfirmPause = async () => {
    if (!selectedDuration) return;

    setIsPausing(true);
    try {
      await pauseProtection(selectedDuration);
      onOpenChange(false);
      setSelectedDuration(null);
    } finally {
      setIsPausing(false);
    }
  };

  const handleCancel = () => {
    onOpenChange(false);
    setSelectedDuration(null);
  };

  const handleQuickAllow = async (appName: string, hostname: string, durationMinutes: number) => {
    const key = hostname || appName;
    setAllowingKey(key);
    try {
      await addToWhitelist(appName, hostname, durationMinutes);
      onOpenChange(false);
    } finally {
      setAllowingKey(null);
    }
  };

  // Get display name for a blocked item
  const getDisplayName = (item: (typeof blockedItems)[0]) => {
    const app = item.usage.application;
    if (app?.hostname) return app.hostname;
    if (app?.name) return app.name;
    return "Unknown";
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-sm border-border bg-background backdrop-blur-xl shadow-xl shadow-black/20"
        showCloseButton={false}
      >
        <DialogHeader className="sr-only">
          <DialogTitle>Pause Protection</DialogTitle>
          <DialogDescription>Choose to pause protection globally or allow a specific app.</DialogDescription>
        </DialogHeader>

        <DialogBody className="space-y-4 pt-6">
          {/* Top Half: Selective Pause */}
          {blockedItems.length > 0 && (
            <>
              <div className="space-y-2">
                <div className="px-1 text-center pb-1">
                  <h3 className="text-[10px] text-muted-foreground font-bold uppercase tracking-wider">
                    Quick allow blocked apps
                  </h3>
                </div>

                <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/5 overflow-hidden divide-y divide-emerald-500/10">
                  {blockedItems.map((item) => {
                    const app = item.usage.application;
                    const key = app?.hostname || app?.bundle_id || String(item.usage.id);
                    const isAllowing = allowingKey === key;

                    return (
                      <div
                        key={key}
                        className="flex items-center gap-3 px-3 py-2.5"
                      >
                        {/* App Icon */}
                        {app?.icon ? (
                          <img
                            src={`data:image/png;base64,${app.icon}`}
                            alt=""
                            className="w-5 h-5 rounded grayscale-[0.2]"
                          />
                        ) : (
                          <div className="w-5 h-5 rounded bg-muted-foreground/20" />
                        )}

                        {/* App Name */}
                        <span className="text-xs font-medium text-muted-foreground flex-1 truncate">
                          {getDisplayName(item)}
                        </span>

                        {/* Allow Buttons */}
                        <div className="flex items-center gap-2">
                          <span className="text-[10px] text-muted-foreground/60 font-medium whitespace-nowrap">Allow for</span>
                          <div className="flex items-center rounded-md border border-border/40 bg-muted/30 overflow-hidden divide-x divide-border/40">
                            {[15, 30, 60].map((durationFn) => (
                              <button
                                key={durationFn}
                                onClick={() =>
                                  handleQuickAllow(app?.executable_path || "", app?.hostname || "", durationFn)
                                }
                                disabled={isAllowing}
                                className="px-2.5 py-1 text-[11px] font-medium text-muted-foreground hover:bg-green-500/10 hover:text-green-400 transition-all disabled:opacity-50"
                              >
                                {isAllowing ? (
                                  "..."
                                ) : (
                                  durationFn === 60 ? "1h" : `${durationFn}m`
                                )}
                              </button>
                            ))}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="h-px w-full bg-border/20 !my-4" />
            </>
          )}

          {/* Bottom Half: Global Pause */}
          <div className="space-y-4">
            <div className="flex flex-col gap-3 px-1">
              <div className="flex items-center gap-2">
                <div className="flex items-center justify-center w-8 h-8 rounded-full bg-orange-500/10 border border-orange-500/20">
                  <IconShieldOff className="w-4 h-4 text-orange-500" />
                </div>
                <h3 className="text-base font-semibold text-foreground">Pause All Protection</h3>
              </div>

              {(todayCount > 0 || weekCount > 0) && (
                <div className="flex items-center gap-1.5 text-xs text-orange-500/90 bg-orange-500/10 border border-orange-500/20 rounded-md px-2.5 py-1.5 w-fit">
                  <IconAlertTriangle className="w-3.5 h-3.5 min-w-[14px]" />
                  <span>
                    You've paused <strong className="font-bold text-orange-500">{todayCount} time{todayCount !== 1 ? "s" : ""}</strong> today, and <strong className="font-bold text-orange-500">{weekCount} time{weekCount !== 1 ? "s" : ""}</strong> this week
                  </span>
                </div>
              )}
            </div>

            {/* Duration Selection */}
            <div className="space-y-2">
              <div className="grid grid-cols-3 gap-2">
                {PAUSE_OPTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    onClick={() => setSelectedDuration(opt.value)}
                    className={`flex flex-col items-center gap-1.5 p-3 rounded-xl border transition-all cursor-pointer ${selectedDuration === opt.value
                      ? "border-orange-500 bg-orange-500/10 text-orange-600 dark:text-orange-400"
                      : "border-border/60 hover:border-border text-muted-foreground hover:bg-muted/50"
                      }`}
                  >
                    <IconClock className="w-4 h-4" />
                    <span className="text-xs font-semibold">{opt.label}</span>
                  </button>
                ))}
              </div>
            </div>
          </div>
        </DialogBody>

        <DialogFooter className="gap-2 pt-2">
          <Button
            variant="ghost"
            onClick={handleCancel}
            className="flex-1 rounded-xl h-11 font-medium hover:bg-muted/80"
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={handleConfirmPause}
            disabled={!selectedDuration || isPausing}
            className={`flex-1 rounded-xl h-11 font-semibold transition-all shadow-sm ${selectedDuration
              ? "bg-orange-500 text-white hover:bg-orange-600 shadow-orange-500/10"
              : "bg-muted text-muted-foreground opacity-50"
              }`}
          >
            {isPausing ? "Pausing..." : "Pause Protection"}
          </Button>
        </DialogFooter>

        {/* Bottom Tip */}
        <div className="px-6 py-3 border-t border-border/20 bg-muted/20 flex items-center justify-center gap-1.5">
          <IconBulb className="w-3 h-3 text-blue-500/60" />
          <p className="text-[10px] text-muted-foreground/80 leading-none">
            Tip:{" "}
            <button
              onClick={() => {
                navigate({ to: "/settings", search: { tab: "rules" } });
                onOpenChange(false);
              }}
              className="text-blue-500/80 hover:text-blue-500 hover:underline transition-colors font-medium"
            >
              Customize your rules
            </button>{" "}
            to allow flexible usage patterns.
          </p>
        </div>
      </DialogContent >
    </Dialog >
  );
}
