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
