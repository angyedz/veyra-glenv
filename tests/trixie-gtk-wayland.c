/*
 * Smoke-test a REAL GTK3 window in a Debian glibc process communicating
 * with an Alpine/musl Weston headless compositor. No rendering assertion:
 * a compositor acceptance and GTK event dispatch test, not GPU benchmark.
 */
#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

typedef int (*gtk_init_check_fn)(int *, char ***);
typedef void *(*gtk_window_new_fn)(int);
typedef void (*gtk_widget_show_all_fn)(void *);
typedef void (*gtk_widget_destroy_fn)(void *);
typedef int (*gtk_events_pending_fn)(void);
typedef int (*gtk_main_iteration_do_fn)(int);

int main(int argc, char **argv) {
    void *gtk = dlopen("libgtk-3.so.0", RTLD_NOW | RTLD_LOCAL);
    if (!gtk) {
        fprintf(stderr, "Cannot load Debian GTK: %s\n", dlerror());
        return EXIT_FAILURE;
    }
    gtk_init_check_fn init_check = (gtk_init_check_fn)dlsym(gtk, "gtk_init_check");
    gtk_window_new_fn window_new = (gtk_window_new_fn)dlsym(gtk, "gtk_window_new");
    gtk_widget_show_all_fn show_all = (gtk_widget_show_all_fn)dlsym(gtk, "gtk_widget_show_all");
    gtk_widget_destroy_fn destroy = (gtk_widget_destroy_fn)dlsym(gtk, "gtk_widget_destroy");
    gtk_events_pending_fn pending = (gtk_events_pending_fn)dlsym(gtk, "gtk_events_pending");
    gtk_main_iteration_do_fn iterate = (gtk_main_iteration_do_fn)dlsym(gtk, "gtk_main_iteration_do");
    if (!init_check || !window_new || !show_all || !destroy || !pending || !iterate) {
        fprintf(stderr, "Required GTK3 functions missing\n");
        return EXIT_FAILURE;
    }
    /* This catches missing image plugins before GTK's fatal fallback path. */
    void *pixbuf = dlopen("libgdk_pixbuf-2.0.so.0", RTLD_NOW | RTLD_LOCAL);
    if (!pixbuf) {
        fprintf(stderr, "Cannot load Debian gdk-pixbuf: %s\n", dlerror());
        return EXIT_FAILURE;
    }
    struct glist { void *data; struct glist *next; };
    struct glist *(*get_formats)(void) =
        (struct glist *(*)(void))dlsym(pixbuf, "gdk_pixbuf_get_formats");
    const char *(*get_format_name)(void *) =
        (const char *(*)(void *))dlsym(pixbuf, "gdk_pixbuf_format_get_name");
    if (!get_formats || !get_format_name) {
        fprintf(stderr, "Debian gdk-pixbuf format discovery API missing\n");
        return EXIT_FAILURE;
    }
    int png_found = 0;
    for (struct glist *it = get_formats(); it; it = it->next) {
        const char *format = get_format_name(it->data);
        if (format) {
            fprintf(stderr, "Debian gdk-pixbuf registered: %s\n", format);
            if (format[0] == 'p' && format[1] == 'n' && format[2] == 'g' &&
                format[3] == '\0') {
                png_found = 1;
            }
        }
    }
    if (!png_found) {
        fprintf(stderr, "Debian gdk-pixbuf PNG decoder was not registered\n");
        return EXIT_FAILURE;
    }
    if (!init_check(&argc, &argv)) {
        fprintf(stderr, "GTK could not initialize its Wayland display\n");
        return EXIT_FAILURE;
    }
    void *window = window_new(0); /* GTK_WINDOW_TOPLEVEL */
    if (!window) {
        fprintf(stderr, "Failed to create GtkWindow\n");
        return EXIT_FAILURE;
    }
    show_all(window);
    for (int i = 0; i < 10; ++i) {
        while (pending()) {
            (void)iterate(0);
        }
        usleep(20000);
    }
    destroy(window);
    while (pending()) {
        (void)iterate(0);
    }
    puts("OK Debian-glibc GTK window created through musl Wayland compositor");
    dlclose(gtk);
    return EXIT_SUCCESS;
}
