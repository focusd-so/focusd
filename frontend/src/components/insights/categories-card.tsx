import { IconFolder, IconArrowRight } from "@tabler/icons-react";
import { Link } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDuration } from "@/lib/mock-data";

interface CategoriesCardProps {
  projects: { name: string; duration_seconds: number }[];
}

const projectColors = [
  "bg-emerald-500",
  "bg-blue-500",
  "bg-amber-500",
  "bg-purple-500",
  "bg-pink-500",
  "bg-cyan-500",
];

export function CategoriesCard({ projects }: CategoriesCardProps) {
  const totalSeconds = projects.reduce((sum, p) => sum + p.duration_seconds, 0);
  const maxSeconds = Math.max(...projects.map((p) => p.duration_seconds), 1);

  const topProjects = projects.slice(0, 3);

  return (
    <Card className="border-border/50">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconFolder className="w-4 h-4 text-muted-foreground" />
            Projects
          </CardTitle>
          <Link
            to="/screen-time/screentime"
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {formatDuration(totalSeconds)} total
            <IconArrowRight className="w-3 h-3" />
          </Link>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {topProjects.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">
            No project activity
          </p>
        ) : (
          topProjects.map((project, index) => {
            const widthPct = (project.duration_seconds / maxSeconds) * 100;
            const colorClass = projectColors[index % projectColors.length];
            return (
              <div key={index} className="space-y-1.5">
                <div className="flex items-center justify-between text-xs">
                  <span className="flex items-center gap-2">
                    <span
                      className={`w-2 h-2 rounded-full ${colorClass}`}
                    />
                    <span className="truncate max-w-[140px] font-medium">
                      {project.name}
                    </span>
                  </span>
                  <span className="text-muted-foreground">
                    {formatDuration(project.duration_seconds)}
                  </span>
                </div>
                <div className="h-2 bg-muted/30 rounded-full overflow-hidden">
                  <div
                    className={`h-full ${colorClass}/70 rounded-full transition-all`}
                    style={{ width: `${widthPct}%` }}
                  />
                </div>
              </div>
            );
          })
        )}
      </CardContent>
    </Card>
  );
}
