import { useState, useMemo, useEffect } from "react";
import { format } from "date-fns";
import { IconCalendar } from "@tabler/icons-react";
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


interface AllowCustomDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (durationMinutes: number) => void;
  appName: string;
  defaultDate?: Date;
}

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

export function AllowCustomDialog({
  open,
  onOpenChange,
  onConfirm,
  appName,
  defaultDate,
}: AllowCustomDialogProps) {
  const [allowUntilDate, setAllowUntilDate] = useState<Date | undefined>(defaultDate || new Date());
  const [allowUntilTime, setAllowUntilTime] = useState<string>("");

  useEffect(() => {
    if (open) {
      const initDate = defaultDate || new Date();
      setAllowUntilDate(initDate);
      setAllowUntilTime(getNextQuarterHourValue(initDate));
    }
  }, [open, defaultDate]);

  const allowUntilDateTime = useMemo(() => {
    if (!allowUntilDate || !allowUntilTime) return null;
    const [hours, minutes] = allowUntilTime.split(":").map(Number);

    if (Number.isNaN(hours) || Number.isNaN(minutes)) return null;

    const date = new Date(allowUntilDate);
    date.setHours(hours, minutes, 0, 0);
    return date;
  }, [allowUntilDate, allowUntilTime]);

  const allowUntilDurationMinutes = useMemo(() => {
    if (!allowUntilDateTime) return null;
    return Math.ceil((allowUntilDateTime.getTime() - Date.now()) / (1000 * 60));
  }, [allowUntilDateTime]);

  const canConfirm = allowUntilDurationMinutes !== null && allowUntilDurationMinutes > 0;

  const handleConfirm = () => {
    if (allowUntilDurationMinutes && allowUntilDurationMinutes > 0) {
      onConfirm(allowUntilDurationMinutes);
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-sm border-border bg-background backdrop-blur-xl shadow-xl shadow-black/20"
        showCloseButton={true}
      >
        <DialogHeader className="space-y-1">
          <DialogTitle>Allow {appName}</DialogTitle>
          <DialogDescription>
            Select a custom date and time to allow access until.
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="space-y-4 pt-4">
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
                      {allowUntilDate ? format(allowUntilDate, "MMM d, yyyy") : "Select date"}
                    </span>
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={allowUntilDate}
                    onSelect={setAllowUntilDate}
                    disabled={(date) => date < new Date(new Date().setHours(0, 0, 0, 0))}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>

              <Select value={allowUntilTime} onValueChange={setAllowUntilTime}>
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

            {allowUntilDurationMinutes !== null && allowUntilDurationMinutes > 0 && (
              <p className="text-[11px] text-muted-foreground">
                Access allowed for {formatPauseDuration(allowUntilDurationMinutes)}
                {allowUntilDateTime ? ` · ${format(allowUntilDateTime, "MMM d, yyyy h:mm a")}` : ""}.
              </p>
            )}

            {allowUntilDurationMinutes !== null && allowUntilDurationMinutes <= 0 && (
              <p className="text-[11px] text-destructive">
                Please select a future date and time.
              </p>
            )}
          </div>
        </DialogBody>

        <DialogFooter className="gap-2 pt-2">
          <Button
            variant="ghost"
            onClick={() => onOpenChange(false)}
            className="flex-1 rounded-xl h-11 font-medium hover:bg-muted/80"
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={handleConfirm}
            disabled={!canConfirm}
            className={`flex-1 rounded-xl h-11 font-semibold transition-all shadow-sm ${canConfirm
              ? "bg-yellow-500 text-yellow-950 hover:bg-yellow-600 shadow-yellow-500/10"
              : "bg-muted text-muted-foreground opacity-50"
              }`}
          >
            Confirm
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
