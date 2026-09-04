//go:build darwin

package main

/*
#cgo LDFLAGS: -framework Carbon -framework CoreFoundation
#include <Carbon/Carbon.h>
#include <CoreFoundation/CoreFoundation.h>
#include <string.h>

static int selectEnglishInputSource(void) {
    // 优先精确选择 ABC。
    const void *keys[] = { kTISPropertyInputSourceID };
    const void *vals[] = { CFSTR("com.apple.keylayout.ABC") };
    CFDictionaryRef filter = CFDictionaryCreate(kCFAllocatorDefault, keys, vals, 1,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    if (filter) {
        CFArrayRef list = TISCreateInputSourceList(filter, false);
        CFRelease(filter);
        if (list && CFArrayGetCount(list) > 0) {
            TISInputSourceRef source = (TISInputSourceRef)CFArrayGetValueAtIndex(list, 0);
            OSStatus st = TISSelectInputSource(source);
            CFRelease(list);
            if (st == noErr) return 1;
        } else if (list) {
            CFRelease(list);
        }
    }

    // ABC 未启用时，选择任意“已启用且 ASCII capable”的键盘布局（如 U.S.）。
    CFArrayRef all = TISCreateInputSourceList(NULL, false);
    if (!all) return 0;
    CFIndex n = CFArrayGetCount(all);
    for (CFIndex i = 0; i < n; i++) {
        TISInputSourceRef src = (TISInputSourceRef)CFArrayGetValueAtIndex(all, i);
        CFBooleanRef enabled = (CFBooleanRef)TISGetInputSourceProperty(src, kTISPropertyInputSourceIsEnabled);
        CFBooleanRef asciiCapable = (CFBooleanRef)TISGetInputSourceProperty(src, kTISPropertyInputSourceIsASCIICapable);
        CFStringRef category = (CFStringRef)TISGetInputSourceProperty(src, kTISPropertyInputSourceCategory);
        if (enabled == kCFBooleanTrue && asciiCapable == kCFBooleanTrue &&
            category && CFStringCompare(category, kTISCategoryKeyboardInputSource, 0) == kCFCompareEqualTo) {
            if (TISSelectInputSource(src) == noErr) {
                CFRelease(all);
                return 1;
            }
        }
    }
    CFRelease(all);
    return 0;
}
*/
import "C"

func switchToEnglishInput() bool {
	return C.selectEnglishInputSource() == 1
}
