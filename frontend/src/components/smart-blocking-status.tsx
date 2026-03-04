import { useState, useRef } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import {
  IconShield,
  IconPlayerPause,
  IconPlayerPlay,
  IconAdjustments,
} from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { useUsageStore } from "@/stores/usage-store";
import { PauseConfirmationDialog } from "@/components/pause-confirmation-dialog";

function formatRemainingTime(seconds: number): string {
  if (seconds <= 0) return "0:00";
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, "0")}`;
}

export function SmartBlockingStatus() {
  const { currentPause, resumeProtection, getPauseHistory, initProtectionStore } =
    useUsageStore();
  const navigate = useNavigate();
  const [showPauseDialog, setShowPauseDialog] = useState(false);
  const hasTriggeredExpiration = useRef(false);

  const handleOpenPauseDialog = () => {
    getPauseHistory(7); // Fetch last 7 days of pause history
    setShowPauseDialog(true);
  };

  const isPaused = !!(currentPause && currentPause.id > 0);

  // Reset expiration flag when pause changes
  if (!isPaused) {
    hasTriggeredExpiration.current = false;
  }

  // Use React Query to poll current time every second when paused
  const { data: currentTime = Date.now() } = useQuery<number>({
    queryKey: ["currentTime", currentPause?.id],
    queryFn: () => Date.now(),
    enabled: isPaused,
    refetchInterval: isPaused ? 1000 : false,
    staleTime: 0,
  });

  // Calculate remaining seconds
  const remainingSeconds = isPaused
    ? (currentPause?.resumed_at ?? 0) - Math.floor(currentTime / 1000)
    : 0;

  // When timer expires, re-fetch protection status from backend to sync state
  if (isPaused && remainingSeconds <= 0 && !hasTriggeredExpiration.current) {
    hasTriggeredExpiration.current = true;
    initProtectionStore();
  }

  const navigateToCustomise = () => {
    navigate({ to: "/settings", search: { tab: "rules" } });
  };

  // Paused state
  if (isPaused) {
    return (
      <div className="p-4 rounded-xl border border-yellow-500/20 bg-yellow-500/5 flex flex-row items-center justify-between gap-4 transition-all">
        <div className="flex items-center gap-3">
          <div className="relative flex items-center justify-center w-10 h-10">
            <div className="relative flex items-center justify-center w-8 h-8 bg-yellow-500/20 rounded-full border border-yellow-500/30">
              <IconPlayerPause className="w-5 h-5 text-yellow-500" />
            </div>
          </div>
          <div className="flex flex-col">
            <div className="flex items-center gap-2">
              <span className="text-sm font-semibold text-yellow-500">
                Paused
              </span>
              <span className="text-sm font-mono font-semibold text-yellow-500">
                {formatRemainingTime(Math.max(0, remainingSeconds))}
              </span>
            </div>
            <span className="text-[10px] text-yellow-500/80">
              Blocking is temporarily suspended
            </span>
          </div>
        </div>

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            className="bg-yellow-500/10 border-yellow-500/30 hover:bg-yellow-500/20 text-yellow-500 text-xs h-8 gap-2"
            onClick={() => resumeProtection()}
          >
            <IconPlayerPlay className="w-3 h-3" />
            Resume
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="border-yellow-500/30 hover:bg-yellow-500/10 hover:text-yellow-500 text-yellow-500 text-xs h-8"
            onClick={navigateToCustomise}
          >
            Customise
          </Button>
        </div>
      </div>
    );
  }

  // Active state (Default)
  return (
    <>
      <div className="p-4 rounded-xl border border-green-500/20 bg-green-500/5 flex flex-row items-center justify-between gap-4 transition-all hover:bg-green-500/[0.07] group/status">
        <div className="flex items-center gap-3">
          <div className="relative flex items-center justify-center w-10 h-10">
            <span className="absolute inline-flex w-full h-full rounded-full opacity-20 animate-pulse bg-green-500"></span>
            <span className="absolute inline-flex w-full h-full rounded-full opacity-40 animate-ping bg-green-500 duration-2000"></span>
            <div className="relative flex items-center justify-center w-8 h-8 bg-green-500 rounded-full shadow-[0_0_12px_rgba(34,197,94,0.4)]">
              <IconShield className="w-5 h-5 text-white" />
            </div>
          </div>
          <div className="flex flex-col">
            <span className="text-sm font-semibold text-green-500 tracking-tight">
              Focus Protection: Active
            </span>
            <span className="text-[10px] text-green-500/70 font-medium">
              Your focus is protected
            </span>
          </div>
        </div>

        <div className="flex gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="hover:bg-orange-500/5 hover:text-orange-400 text-muted-foreground/40 text-xs h-8 gap-2 transition-all"
            onClick={handleOpenPauseDialog}
          >
            <IconPlayerPause className="w-3 h-3" />
            Pause
          </Button>

          <Button
            variant="outline"
            size="sm"
            className="bg-green-500/5 border-green-500/20 hover:bg-green-500/15 hover:text-green-500 hover:border-green-500/40 text-green-500 text-xs h-8 gap-2"
            onClick={navigateToCustomise}
          >
            <IconAdjustments className="w-3 h-3" />
            Custom Rules
          </Button>
        </div>
      </div>

      <PauseConfirmationDialog
        open={showPauseDialog}
        onOpenChange={setShowPauseDialog}
      />
    </>
  );
}
