package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/u-root/u-root/pkg/smbios"
)

type DMIData struct {
	SystemUUID            string
	SystemSerialNumber    string
	BaseboardManufacturer string
	BaseboardProduct      string
	BaseboardSerialNumber string
	ChassisSerialNumber   string
}

func readDMI() (*DMIData, error) {
	sysfsPath := "/sys/firmware/dmi/tables"
	entry, err := ioutil.ReadFile(filepath.Join(sysfsPath, "smbios_entry_point"))
	if err != nil {
		return nil, err
	}
	table, err := ioutil.ReadFile(filepath.Join(sysfsPath, "DMI"))
	if err != nil {
		return nil, err
	}

	si, err := smbios.ParseInfo(entry, table)
	if err != nil {
		return nil, err
	}

	dmi := &DMIData{}
	for _, t := range si.Tables {
		pt, err := smbios.ParseTypedTable(t)
		if err != nil {
			continue
		}
		if ci, ok := pt.(*smbios.ChassisInfo); ok {
			dmi.ChassisSerialNumber = ci.SerialNumber
		} else if bi, ok := pt.(*smbios.BaseboardInfo); ok {
			dmi.BaseboardManufacturer = bi.Manufacturer
			dmi.BaseboardProduct = bi.Product
			dmi.BaseboardSerialNumber = bi.SerialNumber
		} else if si, ok := pt.(*smbios.SystemInfo); ok {
			dmi.SystemSerialNumber = si.SerialNumber
			dmi.SystemUUID = si.UUID.String()
		}
	}

	return dmi, nil
}
