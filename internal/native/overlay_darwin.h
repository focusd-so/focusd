#ifndef OVERLAY_H
#define OVERLAY_H

void ShowOverlay(const char* appName, const char* subtitle, int secondsRemaining);
void UpdateOverlay(int secondsRemaining);
void HideOverlay();

#endif
