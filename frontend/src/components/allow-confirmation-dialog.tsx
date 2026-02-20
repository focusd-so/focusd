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
  { label: "5 min", value: 5 },
  { label: "10 min", value: 10 },
  { label: "15 min", value: 15 },
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
        className="max-w-sm border-yellow-500/20 bg-background/95 backdrop-blur-md"
        showCloseButton={false}
      >
        <DialogHeader className="pb-2">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 rounded-full bg-yellow-500/10 border border-yellow-500/20 overflow-hidden">
              {appIcon ? (
                <img
                  src={
                    appIcon.startsWith("data:")
                      ? appIcon
                      : `data:image/png;base64,${appIcon}`
                  }
                  alt={appName}
                  className="w-8 h-8 object-contain"
                />
              ) : (
                <IconShieldCheck className="w-5 h-5 text-yellow-500" />
              )}
            </div>
            <div>
              <DialogTitle className="text-base">
                Allow Temporarily?
              </DialogTitle>
              <DialogDescription className="text-xs">
                Allow access to {appName}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <DialogBody className="space-y-4">
          {/* Duration Selection */}
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground font-medium">
              Allow duration:
            </p>
            <div className="grid grid-cols-3 gap-2">
              {ALLOW_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setSelectedDuration(opt.value)}
                  className={`flex flex-col items-center gap-1 p-2 rounded-lg border transition-all ${
                    selectedDuration === opt.value
                      ? "border-yellow-500/50 bg-yellow-500/10 text-yellow-400"
                      : "border-border/50 hover:border-yellow-500/30 hover:bg-yellow-500/5 text-muted-foreground hover:text-foreground"
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
            onClick={handleConfirmAllow}
            disabled={!selectedDuration || isAllowing}
            className={`flex-1 transition-all ${
              selectedDuration
                ? "border-yellow-500/50 bg-yellow-500/10 text-yellow-400 hover:bg-yellow-500/20 hover:text-yellow-300"
                : "opacity-50"
            }`}
          >
            {isAllowing ? "Allowing..." : "Allow"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
