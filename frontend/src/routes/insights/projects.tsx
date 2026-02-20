import { createFileRoute } from "@tanstack/react-router";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import {
  mockProjects,
  formatMinutes,
  formatRelativeTime,
} from "@/lib/mock-data";

export const Route = createFileRoute("/insights/projects")({
  component: ProjectsPage,
});

function ProjectsPage() {
  const totalMinutes = mockProjects.reduce((sum, p) => sum + p.totalMinutes, 0);
  const maxMinutes = Math.max(...mockProjects.map((p) => p.totalMinutes));

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Projects</h1>
          <p className="text-sm text-muted-foreground">
            Time tracked across {mockProjects.length} detected projects
          </p>
        </div>
        <div className="text-right">
          <p className="text-2xl font-bold">{formatMinutes(totalMinutes)}</p>
          <p className="text-xs text-muted-foreground">Total today</p>
        </div>
      </div>

      <div className="space-y-3">
        {mockProjects.map((project) => (
          <Card
            key={project.id}
            className="hover:border-border transition-colors"
          >
            <CardContent className="py-4">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-cyan-500/20 to-blue-500/20 flex items-center justify-center">
                    <span className="text-lg">📁</span>
                  </div>
                  <div>
                    <p className="font-semibold">{project.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {project.sessionsCount} sessions · Last active{" "}
                      {formatRelativeTime(project.lastActive)}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-lg font-bold">
                    {formatMinutes(project.totalMinutes)}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {Math.round((project.totalMinutes / totalMinutes) * 100)}%
                    of total
                  </p>
                </div>
              </div>
              <Progress
                value={(project.totalMinutes / maxMinutes) * 100}
                className="h-2"
              />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
