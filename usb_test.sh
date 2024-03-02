sudo mount /dev/sda /mnt
echo "[INFO] usb mounted"
cp main /mnt/main
echo "[INFO] program copied"
sudo umount /mnt
echo "[INFO] usb unmounted"
