import { useState, useMemo } from "react";
import { Link } from "@tanstack/react-router";
import {
  IconAlertTriangle,
  IconCheck,
  IconChevronRight,
  IconClock,
  IconShieldOff,
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
  { label: "5 min", value: 5 },
  { label: "15 min", value: 15 },
  { label: "30 min", value: 30 },
  { label: "1 hour", value: 60 },
];

export function PauseConfirmationDialog({
  open,
  onOpenChange,
}: PauseConfirmationDialogProps) {
  const { getBlockedItemsList, addToWhitelist, pauseProtection, pauseHistory } = useUsageStore();
  const [selectedDuration, setSelectedDuration] = useState<number | null>(null);
  const [isPausing, setIsPausing] = useState(false);
  const [allowingKey, setAllowingKey] = useState<string | null>(null);

  // Get last 2 blocked items to show as quick allow options
  const blockedItems = getBlockedItemsList().slice(0, 2);
  const blockedCount = getBlockedItemsList().length;

  // Calculate pause count from backend history (last 7 days)
  const pauseCount = useMemo(() => {
    if (!pauseHistory) return 0;
    const oneWeekAgoSeconds = Math.floor(Date.now() / 1000) - 7 * 24 * 60 * 60;
    return pauseHistory.filter((p) => p.created_at >= oneWeekAgoSeconds).length;
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

  const handleQuickAllow = async (executablePath: string, hostname: string) => {
    const key = hostname || executablePath;
    setAllowingKey(key);
    try {
      await addToWhitelist(executablePath, hostname, 15); // 15 minutes default
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
        className="max-w-sm border-orange-500/20 bg-background/95 backdrop-blur-md"
        showCloseButton={false}
      >
        <DialogHeader className="pb-2">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 rounded-full bg-orange-500/10 border border-orange-500/20">
              <IconShieldOff className="w-5 h-5 text-orange-500" />
            </div>
            <div>
              <DialogTitle className="text-base">
                Pause All Protection?
              </DialogTitle>
              <DialogDescription className="text-xs">
                This will disable blocking for everything
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <DialogBody className="space-y-4">
          {/* Warning/Stats Section */}
          {pauseCount > 0 && (
            <div className="flex items-center gap-2 text-xs text-orange-400">
              <IconAlertTriangle className="w-3.5 h-3.5" />
              <span>
                You've paused{" "}
                <span className="font-bold">{pauseCount} time{pauseCount !== 1 ? "s" : ""}</span>{" "}
                this week
              </span>
            </div>
          )}

          {/* Quick Allow Section */}
          <div className="rounded-lg border border-green-500/20 bg-green-500/5 overflow-hidden">
            {/* Header */}
            <div className="px-3 py-2 border-b border-green-500/10">
              <p className="text-xs font-medium text-green-400">
                Need just one app?
              </p>
              <p className="text-[10px] text-green-400/60">
                Allow specific items instead of pausing everything
              </p>
            </div>

            {/* Blocked Items List */}
            {blockedItems.length > 0 ? (
              <div className="divide-y divide-green-500/10">
                {blockedItems.map((item) => {
                  const app = item.usage.application;
                  const key = app?.hostname || app?.bundle_id || String(item.usage.id);
                  const isAllowing = allowingKey === key;

                  return (
                    <div
                      key={key}
                      className="flex items-center gap-2 px-3 py-2"
                    >
                      {/* App Icon */}
                      {app?.icon ? (
                        <img
                          src={`data:image/png;base64,${app.icon}`}
                          alt=""
                          className="w-5 h-5 rounded"
                        />
                      ) : (
                        <div className="w-5 h-5 rounded bg-muted-foreground/20" />
                      )}

                      {/* App Name */}
                      <span className="text-xs text-foreground flex-1 truncate">
                        {getDisplayName(item)}
                      </span>

                      {/* Allow Button */}
                      <button
                        onClick={() =>
                          handleQuickAllow(app?.executable_path || "", app?.hostname || "")
                        }
                        disabled={isAllowing}
                        className="flex items-center gap-1 px-2 py-1 text-[10px] font-medium rounded bg-green-500/10 text-green-400 hover:bg-green-500/20 hover:text-green-300 transition-all disabled:opacity-50"
                      >
                        {isAllowing ? (
                          "Allowing..."
                        ) : (
                          <>
                            <IconCheck className="w-3 h-3" />
                            Allow 15m
                          </>
                        )}
                      </button>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="px-3 py-2 text-[10px] text-muted-foreground">
                No items currently blocked
              </div>
            )}

            {/* Link to Custom Rules */}
            <Link
              to="/settings"
              search={{ tab: "customise" }}
              onClick={() => onOpenChange(false)}
              className="flex items-center justify-center gap-1.5 mx-2 my-2 px-3 py-1.5 rounded-md bg-green-500/15 border border-green-500/30 hover:bg-green-500/25 hover:border-green-500/50 transition-all group"
            >
              <span className="text-xs font-medium text-green-400 group-hover:text-green-300">
                Set Up Custom Rules
              </span>
              <IconChevronRight className="w-3.5 h-3.5 text-green-400 group-hover:text-green-300" />
            </Link>
          </div>

          {/* Duration Selection */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <p className="text-xs text-muted-foreground font-medium">
                Pause duration:
              </p>
              {blockedCount > 0 && (
                <p className="text-[10px] text-muted-foreground/60">
                  {blockedCount} item{blockedCount !== 1 ? "s" : ""} blocked
                </p>
              )}
            </div>
            <div className="grid grid-cols-4 gap-2">
              {PAUSE_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setSelectedDuration(opt.value)}
                  className={`flex flex-col items-center gap-1 p-2 rounded-lg border transition-all ${
                    selectedDuration === opt.value
                      ? "border-orange-500/50 bg-orange-500/10 text-orange-400"
                      : "border-border/50 hover:border-orange-500/30 hover:bg-orange-500/5 text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <IconClock className="w-4 h-4" />
                  <span className="text-xs font-medium">{opt.label}</span>
                </button>
              ))}
            </div>
          </div>
        </DialogBody>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleCancel}
            className="flex-1"
          >
            Cancel
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleConfirmPause}
            disabled={!selectedDuration || isPausing}
            className={`flex-1 transition-all ${
              selectedDuration
                ? "border-orange-500/50 bg-orange-500/10 text-orange-400 hover:bg-orange-500/20 hover:text-orange-300"
                : "opacity-50"
            }`}
          >
            {isPausing ? "Pausing..." : "Pause Protection"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
