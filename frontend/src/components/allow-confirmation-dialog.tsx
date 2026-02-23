import { useState } from "react";
import { IconClock, IconShieldCheck } from "@tabler/icons-react";
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

interface AllowConfirmationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  appName: string;
  appIcon?: string;
  onConfirm: (durationMinutes: number) => Promise<void>;
}

const ALLOW_OPTIONS = [
  { label: "15 min", value: 15 },
  { label: "30 min", value: 30 },
  { label: "1 hour", value: 60 },
];

export function AllowConfirmationDialog({
  open,
  onOpenChange,
  appName,
  appIcon,
  onConfirm,
}: AllowConfirmationDialogProps) {
  const [selectedDuration, setSelectedDuration] = useState<number | null>(null);
  const [isAllowing, setIsAllowing] = useState(false);

  const handleConfirmAllow = async () => {
    if (!selectedDuration) return;

    setIsAllowing(true);
    try {
      await onConfirm(selectedDuration);
      onOpenChange(false);
      setSelectedDuration(null);
    } finally {
      setIsAllowing(false);
    }
  };

  const handleCancel = () => {
    onOpenChange(false);
    setSelectedDuration(null);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-sm border-yellow-500/10 bg-background/80 backdrop-blur-2xl shadow-2xl ring-1 ring-white/5"
        showCloseButton={false}
      >
        <DialogHeader className="pb-6">
          <div className="flex items-center gap-4">
            <div className="flex items-center justify-center w-12 h-12 rounded-2xl bg-linear-to-br from-yellow-500/20 to-yellow-600/10 border border-yellow-500/20 shadow-inner overflow-hidden">
              {appIcon ? (
                <img
                  src={
                    appIcon.startsWith("data:")
                      ? appIcon
                      : `data:image/png;base64,${appIcon}`
                  }
                  alt={appName}
                  className="w-9 h-9 object-contain"
                />
              ) : (
                <IconShieldCheck className="w-6 h-6 text-yellow-500" />
              )}
            </div>
            <div className="space-y-0.5">
              <DialogTitle className="text-xl font-semibold tracking-tight">
                Allow Temporarily?
              </DialogTitle>
              <DialogDescription className="text-sm text-muted-foreground/90">
                Allow access to <span className="text-foreground font-medium">{appName}</span>
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <DialogBody className="space-y-6">
          {/* Duration Selection */}
          <div className="space-y-3">
            <p className="text-xs text-muted-foreground font-semibold uppercase tracking-wider">
              Select duration
            </p>
            <div className="grid grid-cols-3 gap-3">
              {ALLOW_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setSelectedDuration(opt.value)}
                  className={`flex flex-col items-center gap-2 p-3 rounded-xl border transition-colors cursor-pointer ${selectedDuration === opt.value
                    ? "border-yellow-500/50 bg-yellow-500/15 text-yellow-500 shadow-[0_0_15px_rgba(234,179,8,0.1)]"
                    : "border-border/40 hover:border-yellow-500/30 hover:bg-yellow-500/5 text-muted-foreground hover:text-foreground"
                    }`}
                >
                  <IconClock className="w-5 h-5 opacity-80" />
                  <span className="text-xs font-bold leading-none">{opt.label}</span>
                </button>
              ))}
            </div>
          </div>
        </DialogBody>

        <DialogFooter className="gap-3 pt-2">
          <Button
            variant="outline"
            size="lg"
            onClick={handleCancel}
            className="flex-1 rounded-xl border-border/40 hover:bg-muted/50 font-medium"
          >
            Cancel
          </Button>
          <Button
            variant="default"
            size="lg"
            onClick={handleConfirmAllow}
            disabled={!selectedDuration || isAllowing}
            className={`flex-1 rounded-xl font-bold transition-all shadow-lg ${selectedDuration
              ? "bg-yellow-500 text-yellow-950 hover:bg-yellow-400 shadow-yellow-500/20 active:scale-[0.98] border-none"
              : "bg-muted text-muted-foreground opacity-50 gray-scale"
              }`}
          >
            {isAllowing ? "Allowing..." : "Allow"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
