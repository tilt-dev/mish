<img src="./mish.png" width="571" height="180" title="Michel is a hermit crab">

Bonjour! Je m'appelle Michel—pronounced mee-shell (/mi.ʃɛl/). Coincidentally, that's also how you pronounce `mish`, which stands for **Mi**ll **Sh**ell. Feel free to imagine the rest of this document in an outrageous French accent as I help you get `mish` up and running.

I'm an early experiment, not a finished product, but I bet I can still make your workflow a little bit better. Read more [on the Windmill Blog](https://medium.com/windmill-engineering/mish-cruise-control-for-developers-98629709b5ec).

![mish demo gifcast](./gifcast.gif "mish demo gifcast")

## Getting the binary
If you're running OSX or Linux, you can download a pre-compiled binary from [**our releases page**](https://github.com/windmilleng/mish/releases). Otherwise, you can install via the Go toolchain. First [install Go](https://golang.org/doc/install#install) and make sure that you've added `/usr/local/go/bin` to the `PATH` environment variable. Then install `mish` with:
```bash
go get -u github.com/windmilleng/mish/cmd/mish
```

## Using `mish`

Configuration happens in your `notes.mill`. Make one to get started:
```bash
echo "sh(\"echo hello world\")" > notes.mill
```

### Hotkeys
* `↓`/`↑`: scroll down/up
* `j`/`k`: jump down/up one command
* `o`: expand/collapse current command output
* `r`: run your notes.mill file
* `q`: quit

### Available Mill Functions
* `sh`: execute arbitrary shell commands
  * `sh("my shell command")`
  * Normally, when a your shell commands exits with a non-zero status code, `mish` will abort the whole execution. To continue execution if a given command fails, specify `sh("faily command", tolerate_failure=True)`
* Filesystem interaction:
  * `watch`: specify files to watch. Edits to these files will be displayed in the `mish` status bar.
  * `autorun`: specify files that will trigger rerunning of your notes.mill. (Autorun works based on what's been `watch`'ed; you must `watch` a file before `autorun`'ing it.)
  * _patterns_: `watch` and `autorun` each specify files by taking any number of _patterns_. A _pattern_ is a string that matches files.
    * The wildcard `*` expands to any text in a directory. `"foo/*.go"` matches all go files in the directory `foo`
    * The wildcard `**` expands to match any number of directories. `"**/*.go"` matches all go files in this directory or subdirectories.
    * _patterns_ match files, not directories. If `foo` is a directory, `"foo"` will not match any files. (Use `"foo/**"`)
    * If a _pattern_ starts with "!", its an inversion. `watch('**', '!.git/**')` will watch all files, except your `.git` directory.

### Example notes.mill
```python
### Configs
watch("**", "!.git/**")
autorun("./server/*.go", "./common/*/*_test.go")

### Commands to execute
sh("make proto")
sh("go build ./server")
sh("go test server", tolerate_failure=True) # if this exits w/ non-zero code, keep going
sh("go test common")
```

## Tell Us What You Think!
Tried `mish`? Loved it? Hated it? Couldn't get past the install? [**Take our survey**](https://docs.google.com/forms/d/e/1FAIpQLSf8UXLG0FOeMswoW7LuUP02CeUwKBccJishJKDE_VyOqe7g_g/viewform?usp=sf_link) and tell us about your experience. Your feedback will help us make dev tools better for everyone!

### Guiding Questions for Alpha Users
If you're one of our amazing alpha users, thank you! We appreciate you taking the time to test our product and give us feedback. Here are some things that we'd love for you to keep in mind as you test out `mish` so we can pick your brain about them later:
1. When do you find yourself editing notes.mill? If it occurs to you to edit notes.mill and you don’t, why?
2. When are you still using your shell instead of mish? Why?
3. Did you get annoyed/distracted by `mish`, or turn it off?
4. How does writing in Mill (our configuration language) feel? Intuitive? Annoying?
5. How has `mish` changed your workflow? (Alternately: how does it fit into your existing workflow?)
