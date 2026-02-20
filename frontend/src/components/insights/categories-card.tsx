import { IconFolder } from "@tabler/icons-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMinutes, type ProjectStats } from "@/lib/mock-data";

interface CategoriesCardProps {
  projects: ProjectStats[];
}

// Color palette for projects
const projectColors = [
  "bg-emerald-500",
  "bg-blue-500",
  "bg-amber-500",
  "bg-purple-500",
  "bg-pink-500",
  "bg-cyan-500",
];

export function CategoriesCard({ projects }: CategoriesCardProps) {
  const totalMinutes = projects.reduce((sum, p) => sum + p.totalMinutes, 0);
  const maxMinutes = Math.max(...projects.map((p) => p.totalMinutes), 1);

  // Show top 5 projects
  const topProjects = projects.slice(0, 5);

  return (
    <Card className="border-border/50">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <IconFolder className="w-4 h-4 text-muted-foreground" />
            Projects
          </CardTitle>
          <span className="text-xs text-muted-foreground">
            {formatMinutes(totalMinutes)} total
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {topProjects.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-4">
            No project activity
          </p>
        ) : (
          topProjects.map((project, index) => {
            const widthPct = (project.totalMinutes / maxMinutes) * 100;
            const colorClass = projectColors[index % projectColors.length];
            return (
              <div key={project.id} className="space-y-1.5">
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
                    {formatMinutes(project.totalMinutes)}
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
