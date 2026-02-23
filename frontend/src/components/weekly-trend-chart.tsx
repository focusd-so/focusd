import { useMemo } from "react";
import {
  Bar,
  BarChart,
  XAxis,
  YAxis,
  ReferenceLine,
  CartesianGrid,
} from "recharts";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
} from "@/components/ui/chart";

export type DailyStats = {
  date: number;
  productive_minutes: number;
  neutral_minutes: number;
  distractive_minutes: number;
};

const trendChartConfig = {
  productive: { label: "Productive", color: "#22c55e" },
  neutral: { label: "Neutral", color: "#eab308" },
  distractive: { label: "Distractive", color: "#ef4444" },
} satisfies ChartConfig;

interface WeeklyTrendChartProps {
  data?: DailyStats[];
  targetProductiveHours?: number;
}

function generateMockData(): DailyStats[] {
  const now = new Date();
  const data: DailyStats[] = [];

  for (let i = 6; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i);
    date.setHours(0, 0, 0, 0);

    // Generate varied but realistic mock data
    const productiveBase = 240 + Math.floor(Math.random() * 180); // 4-7 hours
    const neutralBase = 30 + Math.floor(Math.random() * 60); // 0.5-1.5 hours
    const distractiveBase = 20 + Math.floor(Math.random() * 80); // 0.3-1.6 hours

    data.push({
      date: Math.floor(date.getTime() / 1000),
      productive_minutes: productiveBase,
      neutral_minutes: neutralBase,
      distractive_minutes: distractiveBase,
    });
  }

  return data;
}

function getDayName(dateUnix: number): string {
  const date = new Date(dateUnix * 1000);
  return date.toLocaleDateString("en-US", { weekday: "short" });
}

function minutesToHours(minutes: number): number {
  return Math.round((minutes / 60) * 10) / 10;
}

function formatHours(hours: number): string {
  return `${hours}h`;
}

export function WeeklyTrendChart({
  data,
  targetProductiveHours = 6,
}: WeeklyTrendChartProps) {
  // Use mock data if no data provided
  const mockData = useMemo(() => generateMockData(), []);
  const displayData = data && data.length > 0 ? data : mockData;

  // Take last 7 days of data
  const last7Days = displayData.slice(-7);

  // Transform data for the chart
  const chartData = last7Days.map((day) => ({
    day: getDayName(day.date),
    productive: minutesToHours(day.productive_minutes),
    neutral: minutesToHours(day.neutral_minutes),
    distractive: minutesToHours(day.distractive_minutes),
  }));

  return (
    <ChartContainer config={trendChartConfig} className="h-[300px] w-full">
      <BarChart
        data={chartData}
        margin={{ top: 10, right: 10, left: 0, bottom: 0 }}
      >
        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
        <XAxis
          dataKey="day"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          className="text-xs"
        />
        <YAxis
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          tickFormatter={formatHours}
          className="text-xs"
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              formatter={(value) => formatHours(Number(value))}
            />
          }
        />
        <ChartLegend content={<ChartLegendContent />} />
        <ReferenceLine
          y={targetProductiveHours}
          stroke="hsl(var(--muted-foreground))"
          strokeDasharray="5 5"
          strokeWidth={2}
          label={{
            value: `Target: ${targetProductiveHours}h`,
            position: "insideTopRight",
            className: "text-xs fill-muted-foreground",
          }}
        />
        <Bar
          dataKey="productive"
          stackId="1"
          fill="var(--color-productive)"
          radius={[0, 0, 0, 0]}
        />
        <Bar
          dataKey="neutral"
          stackId="1"
          fill="var(--color-neutral)"
          radius={[0, 0, 0, 0]}
        />
        <Bar
          dataKey="distractive"
          stackId="1"
          fill="var(--color-distractive)"
          radius={[4, 4, 0, 0]}
        />
      </BarChart>
    </ChartContainer>
  );
}
