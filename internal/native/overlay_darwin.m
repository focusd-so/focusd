#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include "overlay_darwin.h"

void MyFree(void *ptr) {
    free(ptr);
}

// Static reference to our overlay window
static NSPanel* gOverlayPanel = nil;
static NSTextField* gLabel = nil;
static NSTextField* gSubtitleLabel = nil;
static NSString* gCurrentAppName = nil;
static NSString* gCurrentSubtitle = nil;

void UpdateOverlay(int secondsRemaining) {
    if (!gOverlayPanel || !gLabel || !gCurrentAppName) return;
    
    dispatch_async(dispatch_get_main_queue(), ^{
        NSString* text = [NSString stringWithFormat:@"%@ will be blocked in %ds", gCurrentAppName, secondsRemaining];
        [gLabel setStringValue:text];
        
        // Pulse/Urgency effect: Red overlay if < 5s?
        if (secondsRemaining <= 5) {
             [gLabel setTextColor:[NSColor colorWithRed:1.0 green:0.2 blue:0.2 alpha:1.0]];
        }
    });
}

void ShowOverlay(const char* appName, const char* subtitle, int secondsRemaining) {
    if (appName) {
        gCurrentAppName = [NSString stringWithUTF8String:appName];
    } else if (!gCurrentAppName) {
        gCurrentAppName = @"Application";
    }

    if (subtitle) {
        gCurrentSubtitle = [NSString stringWithUTF8String:subtitle];
    }

    if (gOverlayPanel) {
        // Just update subtitle separate from timer update if needed
        dispatch_async(dispatch_get_main_queue(), ^{
             if (gSubtitleLabel && gCurrentSubtitle) {
                 [gSubtitleLabel setStringValue:gCurrentSubtitle];
             }
        });
        UpdateOverlay(secondsRemaining);
        return;
    }

    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindowStyleMask style = NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel;
        
        gOverlayPanel = [[NSPanel alloc] initWithContentRect:NSMakeRect(0, 0, 400, 70) // Increased height for subtitle
                                                   styleMask:style
                                                     backing:NSBackingStoreBuffered
                                                       defer:NO];
        
        [gOverlayPanel setFloatingPanel:YES];
        [gOverlayPanel setLevel:NSStatusWindowLevel]; 
        [gOverlayPanel setHidesOnDeactivate:NO];
        [gOverlayPanel setBackgroundColor:[NSColor clearColor]]; 
        [gOverlayPanel setOpaque:NO];
        [gOverlayPanel setHasShadow:YES];
        [gOverlayPanel setIgnoresMouseEvents:YES];

        // Center on screen
        NSRect screenRect = [[NSScreen mainScreen] visibleFrame];
        CGFloat width = 500; // Wider to fit subtitle
        CGFloat height = 70; 
        CGFloat x = (screenRect.size.width - width) / 2 + screenRect.origin.x;
        CGFloat y = screenRect.origin.y + screenRect.size.height - 130; 
        [gOverlayPanel setFrame:NSMakeRect(x, y, width, height) display:YES];

        // "Urgent" Visual Effect
        NSVisualEffectView* visualEffectView = [[NSVisualEffectView alloc] initWithFrame:NSMakeRect(0, 0, width, height)];
        [visualEffectView setMaterial:NSVisualEffectMaterialHUDWindow]; 
        [visualEffectView setBlendingMode:NSVisualEffectBlendingModeBehindWindow];
        [visualEffectView setState:NSVisualEffectStateActive];
        [visualEffectView setWantsLayer:YES];
        [visualEffectView.layer setCornerRadius:16.0]; 
        [visualEffectView.layer setMasksToBounds:YES];
        [visualEffectView.layer setBorderWidth:1.5];
        [visualEffectView.layer setBorderColor:[[NSColor colorWithRed:1.0 green:0.3 blue:0.3 alpha:0.6] CGColor]]; 
        
        [gOverlayPanel setContentView:visualEffectView];
        
        // Add a red tint view
        NSView* tintView = [[NSView alloc] initWithFrame:visualEffectView.bounds];
        [tintView setWantsLayer:YES];
        [tintView.layer setBackgroundColor:[[NSColor colorWithRed:0.5 green:0.0 blue:0.0 alpha:0.2] CGColor]]; 
        [visualEffectView addSubview:tintView];

        // Main Label (App + Timer)
        gLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(10, 36, width - 20, 24)];
        [gLabel setBezeled:NO];
        [gLabel setDrawsBackground:NO];
        [gLabel setEditable:NO];
        [gLabel setSelectable:NO];
        [gLabel setAlignment:NSTextAlignmentCenter];
        
        CGFloat fontSize = 16.0;
        [gLabel setFont:[NSFont systemFontOfSize:fontSize weight:NSFontWeightBold]]; 
        [gLabel setTextColor:[NSColor whiteColor]];
        [visualEffectView addSubview:gLabel];

        // Subtitle Label (Details)
        gSubtitleLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(10, 10, width - 20, 20)];
        [gSubtitleLabel setBezeled:NO];
        [gSubtitleLabel setDrawsBackground:NO];
        [gSubtitleLabel setEditable:NO];
        [gSubtitleLabel setSelectable:NO];
        [gSubtitleLabel setAlignment:NSTextAlignmentCenter];
        [gSubtitleLabel setFont:[NSFont systemFontOfSize:13.0 weight:NSFontWeightRegular]];
        [gSubtitleLabel setTextColor:[NSColor colorWithWhite:0.9 alpha:1.0]];
        
        if (gCurrentSubtitle) {
            [gSubtitleLabel setStringValue:gCurrentSubtitle];
        }

        [visualEffectView addSubview:gSubtitleLabel];
        
        // Initial update
        UpdateOverlay(secondsRemaining);
        
        [gOverlayPanel orderFront:nil];
    });
}

void HideOverlay() {
    if (!gOverlayPanel) return;
    
    dispatch_async(dispatch_get_main_queue(), ^{
        [gOverlayPanel close];
        gOverlayPanel = nil;
        gLabel = nil;
        gSubtitleLabel = nil;
        
        gCurrentAppName = nil;
        gCurrentSubtitle = nil;
    });
}
