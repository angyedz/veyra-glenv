/* Real glibc Wayland IPC smoke test without host Wayland headers.
 * This connects to a musl-native Weston compositor through Debian's
 * libwayland-client and the original glenv Wayland preload filter.
 */
#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>

typedef void *(*wl_connect_fn)(const char *);
typedef int (*wl_roundtrip_fn)(void *);
typedef int (*wl_get_fd_fn)(void *);
typedef void (*wl_disconnect_fn)(void *);

int main(void) {
    void *lib = dlopen("libwayland-client.so.0", RTLD_NOW | RTLD_LOCAL);
    if (!lib) {
        fprintf(stderr, "Debian libwayland-client missing: %s\n", dlerror());
        return EXIT_FAILURE;
    }
    wl_connect_fn connect_display = (wl_connect_fn)dlsym(lib, "wl_display_connect");
    wl_roundtrip_fn roundtrip = (wl_roundtrip_fn)dlsym(lib, "wl_display_roundtrip");
    wl_get_fd_fn get_fd = (wl_get_fd_fn)dlsym(lib, "wl_display_get_fd");
    wl_disconnect_fn disconnect_display = (wl_disconnect_fn)dlsym(lib, "wl_display_disconnect");
    if (!connect_display || !roundtrip || !get_fd || !disconnect_display) {
        fprintf(stderr, "Debian Wayland ABI incomplete\n");
        return EXIT_FAILURE;
    }
    void *display = connect_display(NULL);
    if (!display) {
        fprintf(stderr, "Failed to connect to musl-native Wayland compositor\n");
        return EXIT_FAILURE;
    }
    int fd = get_fd(display);
    int result = roundtrip(display);
    disconnect_display(display);
    dlclose(lib);
    if (fd < 0 || result < 0) {
        fprintf(stderr, "Wayland roundtrip failed (fd=%d, result=%d)\n", fd, result);
        return EXIT_FAILURE;
    }
    puts("OK real Debian-glibc Wayland roundtrip via musl-native Weston");
    return EXIT_SUCCESS;
}
