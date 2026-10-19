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
#include <unistd.h>

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
    const char *prefix = getenv("GLENV_TRIXIE_ROOT");
    const char *overrides[] = {
        "SPA_PLUGIN_DIR",
        "PIPEWIRE_MODULE_DIR",
        "PIPEWIRE_CONFIG_DIR",
    };
    if (prefix && prefix[0]) {
        for (size_t i = 0; i < sizeof(overrides)/sizeof(overrides[0]); ++i) {
            const char *actual = getenv(overrides[i]);
            char expected[4096];
            if (snprintf(expected, sizeof(expected), "%s/runtime/sysroot/", prefix) < 0) {
                return EXIT_FAILURE;
            }
            if (!actual || strncmp(actual, expected, strlen(expected)) != 0 ||
                access(actual, F_OK) != 0) {
                fprintf(stderr, "FAIL Debian runtime plugin path %s=%s\n",
                        overrides[i], actual ? actual : "(missing)");
                ++failures;
            } else {
                printf("OK Debian plugin path %s=%s\n", overrides[i], actual);
            }
        }
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
