import { useState, useMemo, useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  IconAlertTriangle,
  IconCalendar,
  IconClock,
  IconShieldOff,
  IconBulb,
} from "@tabler/icons-react";
import { format } from "date-fns";
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
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useUsageStore } from "@/stores/usage-store";
import { cn } from "@/lib/utils";

interface PauseConfirmationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PAUSE_OPTIONS = [
  { label: "15 min", value: 15 },
  { label: "30 min", value: 30 },
  { label: "1 hour", value: 60 },
];

const TIME_OPTIONS = Array.from({ length: 96 }, (_, index) => {
  const hours = Math.floor(index / 4);
  const minutes = (index % 4) * 15;
  const value = `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}`;
  const label = new Date(2024, 0, 1, hours, minutes).toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
  });

  return { value, label };
});

function getNextQuarterHourValue(date = new Date()): string {
  const next = new Date(date);
  next.setSeconds(0, 0);
  next.setMinutes(Math.ceil(next.getMinutes() / 15) * 15);

  if (next.getMinutes() === 60) {
    next.setHours(next.getHours() + 1, 0, 0, 0);
  }

  return `${String(next.getHours()).padStart(2, "0")}:${String(next.getMinutes()).padStart(2, "0")}`;
}

function formatPauseDuration(totalMinutes: number): string {
  const days = Math.floor(totalMinutes / (60 * 24));
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60);
  const minutes = totalMinutes % 60;

  const parts: string[] = [];

  if (days > 0) {
    parts.push(`${days} day${days === 1 ? "" : "s"}`);
  }
  if (hours > 0) {
    parts.push(`${hours} hour${hours === 1 ? "" : "s"}`);
  }
  if (days === 0 && minutes > 0) {
    parts.push(`${minutes} minute${minutes === 1 ? "" : "s"}`);
  }

  return parts.join(" ");
}

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
  const [pauseMode, setPauseMode] = useState<"duration" | "until">("duration");
  const [selectedDuration, setSelectedDuration] = useState<number | null>(null);
  const [pauseUntilDate, setPauseUntilDate] = useState<Date | undefined>(undefined);
  const [pauseUntilTime, setPauseUntilTime] = useState<string>("");
  const [isPausing, setIsPausing] = useState(false);
  const [allowingKey, setAllowingKey] = useState<string | null>(null);

  const resetPauseForm = () => {
    setPauseMode("duration");
    setSelectedDuration(null);
    setPauseUntilDate(undefined);
    setPauseUntilTime("");
  };

  // Fetch pause history for the last 30 days when the dialog opens
  // and reset selected duration when the dialog is closed.
  useEffect(() => {
    if (open) {
      getPauseHistory(30);
    } else {
      resetPauseForm();
    }
  }, [open, getPauseHistory]);

  const pauseUntilDateTime = useMemo(() => {
    if (!pauseUntilDate || !pauseUntilTime) return null;
    const [hours, minutes] = pauseUntilTime.split(":").map(Number);

    if (Number.isNaN(hours) || Number.isNaN(minutes)) return null;

    const date = new Date(pauseUntilDate);
    date.setHours(hours, minutes, 0, 0);
    return date;
  }, [pauseUntilDate, pauseUntilTime]);

  const pauseUntilDurationMinutes = useMemo(() => {
    if (!pauseUntilDateTime) return null;
    return Math.ceil((pauseUntilDateTime.getTime() - Date.now()) / (1000 * 60));
  }, [pauseUntilDateTime]);

  const canConfirmPause =
    pauseMode === "duration"
      ? !!selectedDuration
      : !!pauseUntilDurationMinutes && pauseUntilDurationMinutes > 0;

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
    let durationMinutes: number | null = null;

    if (pauseMode === "duration") {
      durationMinutes = selectedDuration;
    } else if (pauseUntilDateTime) {
      durationMinutes = Math.ceil((pauseUntilDateTime.getTime() - Date.now()) / (1000 * 60));
    }

    if (!durationMinutes || durationMinutes <= 0) return;

    setIsPausing(true);
    try {
      await pauseProtection(durationMinutes);
      onOpenChange(false);
      resetPauseForm();
    } finally {
      setIsPausing(false);
    }
  };

  const handleCancel = () => {
    onOpenChange(false);
    resetPauseForm();
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
                    const key = app?.hostname || app?.name || String(item.usage.id);
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
                                  handleQuickAllow(app?.name || "", app?.hostname || "", durationFn)
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
                  <Button
                    key={opt.value}
                    type="button"
                    variant="ghost"
                    onClick={() => {
                      setPauseMode("duration");
                      setSelectedDuration(opt.value);
                    }}
                    className={cn(
                      "h-9 px-2 rounded-lg border transition-all text-xs font-medium gap-1.5",
                      pauseMode === "duration" && selectedDuration === opt.value
                        ? "border-orange-500 bg-orange-500/10 text-orange-600 dark:text-orange-400 hover:bg-orange-500/15"
                        : "border-border/60 hover:border-border text-muted-foreground hover:bg-muted/50"
                    )}
                  >
                    <IconClock className="w-3.5 h-3.5" />
                    <span>{opt.label}</span>
                  </Button>
                ))}
              </div>

              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  if (pauseMode === "until") {
                    setPauseMode("duration");
                    setPauseUntilDate(undefined);
                    setPauseUntilTime("");
                    return;
                  }

                  setPauseMode("until");
                  setSelectedDuration(null);
                  if (!pauseUntilDate) {
                    const now = new Date();
                    setPauseUntilDate(now);
                    setPauseUntilTime(getNextQuarterHourValue(now));
                  }
                }}
                className={cn(
                  "w-full h-9 px-3 rounded-lg border transition-all text-xs font-medium gap-1.5",
                  pauseMode === "until"
                    ? "border-orange-500 bg-orange-500/10 text-orange-600 dark:text-orange-400 hover:bg-orange-500/15"
                    : "border-border/60 hover:border-border text-muted-foreground hover:bg-muted/50"
                )}
              >
                <IconCalendar className="w-3.5 h-3.5" />
                <span>Pause until</span>
              </Button>

              {pauseMode === "until" && (
                <div className="rounded-xl border border-border/60 p-3 space-y-2 bg-muted/20">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    <Popover>
                      <PopoverTrigger asChild>
                        <Button
                          type="button"
                          variant="outline"
                          className="justify-start text-left font-normal w-full min-w-0"
                        >
                          <IconCalendar className="mr-2 h-4 w-4" />
                          <span className="truncate">
                            {pauseUntilDate ? format(pauseUntilDate, "MMM d, yyyy") : "Select date"}
                          </span>
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent className="w-auto p-0" align="start">
                        <Calendar
                          mode="single"
                          selected={pauseUntilDate}
                          onSelect={setPauseUntilDate}
                          disabled={(date) => date < new Date(new Date().setHours(0, 0, 0, 0))}
                          initialFocus
                        />
                      </PopoverContent>
                    </Popover>

                    <Select value={pauseUntilTime} onValueChange={setPauseUntilTime}>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Select time" />
                      </SelectTrigger>
                      <SelectContent>
                        {TIME_OPTIONS.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {pauseUntilDurationMinutes !== null && pauseUntilDurationMinutes > 0 && (
                    <p className="text-[11px] text-muted-foreground">
                      Protection will resume in {formatPauseDuration(pauseUntilDurationMinutes)}
                      {pauseUntilDateTime ? ` · ${format(pauseUntilDateTime, "MMM d, yyyy h:mm a")}` : ""}.
                    </p>
                  )}

                  {pauseUntilDurationMinutes !== null && pauseUntilDurationMinutes <= 0 && (
                    <p className="text-[11px] text-destructive">
                      Please select a future date and time.
                    </p>
                  )}
                </div>
              )}
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
            disabled={!canConfirmPause || isPausing}
            className={`flex-1 rounded-xl h-11 font-semibold transition-all shadow-sm ${canConfirmPause
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
