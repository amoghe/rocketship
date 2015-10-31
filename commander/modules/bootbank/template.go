package bootbank

var (
	grubConfTemplateStr = `set default="{{ .DefaultBootEntry }}"
set timeout=3

set menu_color_normal=white/black
set menu_color_highlight=black/light-gray

if sleep --verbose --interruptible {{ .HiddenTimeout }} ; then
  echo "Loading ..."
  set timeout=0
fi

{{ range .BootbankEntries }}
menuentry "{{ .MenuEntry }}" {
	insmod  ext2
	search  --label --set=root --no-floppy {{ .PartitionLabel }}
	linux   /vmlinuz root=LABEL={{ .PartitionLabel }} {{ .KernelCmdlineOpts }}"
	initrd  /initrd.img
}
{{ end }}
`
)
