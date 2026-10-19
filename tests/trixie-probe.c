/*
 * Real glibc/graphics/audio *library-loading* probe, not a GUI/audio test.
 * Compiled on Ubuntu and executed with Debian Trixie's loader and libraries
 * inside an Alpine Linux (musl) container.
 */
#define _GNU_SOURCE
#include <dlfcn.h>
#include <gnu/libc-version.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

struct library {
    const char *name;
    const char *component;
};
static const struct library required[] = {
    {"libstdc++.so.6", "C++ runtime"},
    {"libwayland-client.so.0", "Wayland client"},
    {"libEGL.so.1", "EGL/GLVND"},
    {"libGL.so.1", "OpenGL/GLVND"},
    {"libpipewire-0.3.so.0", "PipeWire client"},
    {"libgtk-3.so.0", "GTK 3"},
    {"libnss3.so", "NSS TLS"},
};

int main(void) {
    int failures = 0;
    printf("glenv Debian libc version: %s\n", gnu_get_libc_version());
    if (strcmp(gnu_get_libc_version(), "2.41") != 0) {
        fprintf(stderr, "Expected glibc 2.41 from Debian Trixie\n");
        return EXIT_FAILURE;
    }
    for (size_t index = 0; index < sizeof(required) / sizeof(required[0]); ++index) {
        dlerror();
        void *handle = dlopen(required[index].name, RTLD_NOW | RTLD_LOCAL);
        if (!handle) {
            fprintf(stderr, "FAIL %s: %s: %s\n",
                    required[index].component, required[index].name, dlerror());
            ++failures;
            continue;
        }
        printf("OK %s: %s\n", required[index].component, required[index].name);
        dlclose(handle);
    }
    return failures == 0 ? EXIT_SUCCESS : EXIT_FAILURE;
}
