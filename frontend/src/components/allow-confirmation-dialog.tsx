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
        className="max-w-sm border-border bg-background backdrop-blur-xl shadow-xl shadow-black/20"
        showCloseButton={false}
      >
        <DialogHeader className="pb-4">
          <div className="flex flex-col items-center gap-4 text-center">
            {appIcon ? (
              <img
                src={
                  appIcon.startsWith("data:")
                    ? appIcon
                    : `data:image/png;base64,${appIcon}`
                }
                alt={appName}
                className="w-14 h-14 object-contain"
              />
            ) : (
              <IconShieldCheck className="w-10 h-10 text-yellow-500" />
            )}
            <div className="space-y-1">
              <DialogTitle className="text-xl font-semibold">
                Allow Temporarily?
              </DialogTitle>
              <DialogDescription className="text-sm">
                Allow access to <span className="font-medium text-foreground">{appName}</span>
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <DialogBody className="space-y-6">
          <div className="space-y-3">
            <p className="text-[10px] text-muted-foreground font-bold uppercase tracking-wider text-center">
              Duration
            </p>
            <div className="grid grid-cols-3 gap-2">
              {ALLOW_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setSelectedDuration(opt.value)}
                  className={`flex flex-col items-center gap-1.5 p-3 rounded-xl border transition-all cursor-pointer ${selectedDuration === opt.value
                    ? "border-yellow-500 bg-yellow-500/10 text-yellow-600 dark:text-yellow-400"
                    : "border-border/60 hover:border-border text-muted-foreground hover:bg-muted/50"
                    }`}
                >
                  <IconClock className="w-4 h-4" />
                  <span className="text-xs font-semibold">{opt.label}</span>
                </button>
              ))}
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
            onClick={handleConfirmAllow}
            disabled={!selectedDuration || isAllowing}
            className={`flex-1 rounded-xl h-11 font-semibold transition-all shadow-sm ${selectedDuration
              ? "bg-yellow-500 text-white hover:bg-yellow-600 shadow-yellow-500/10"
              : "bg-muted text-muted-foreground opacity-50"
              }`}
          >
            {isAllowing ? "Allowing..." : "Allow"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
