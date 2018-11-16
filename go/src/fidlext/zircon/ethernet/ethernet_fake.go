// Copyright 2018 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ethernet

import (
	"syscall/zx"
	"testing"

	"fidl/zircon/ethernet"
)

var _ ethernet.Device = (*Device)(nil)

type Device struct {
	testing.TB

	GetInfoImpl                           func() (ethernet.Info, error)
	GetFifosImpl                          func() (int32, *ethernet.Fifos, error)
	SetIoBufferImpl                       func(zx.VMO) (int32, error)
	StartImpl                             func() (int32, error)
	StopImpl                              func() error
	ListenStartImpl                       func() (int32, error)
	ListenStopImpl                        func() error
	SetClientNameImpl                     func(string) (int32, error)
	GetStatusImpl                         func() (uint32, error)
	SetPromiscuousModeImpl                func(bool) (int32, error)
	ConfigMulticastAddMacImpl             func(addr ethernet.MacAddress) (int32, error)
	ConfigMulticastDeleteMacImpl          func(addr ethernet.MacAddress) (int32, error)
	ConfigMulticastSetPromiscuousModeImpl func(enabled bool) (int32, error)
	ConfigMulticastTestFilterImpl         func() (int32, error)
	DumpRegistersImpl                     func() (int32, error)
}

func (d *Device) GetInfo() (ethernet.Info, error) {
	fn := d.GetInfoImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to GetInfo")
	}
	return fn()
}

func (d *Device) GetFifos() (int32, *ethernet.Fifos, error) {
	fn := d.GetFifosImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to GetFifos")
	}
	return fn()
}

func (d *Device) SetIoBuffer(h zx.VMO) (int32, error) {
	fn := d.SetIoBufferImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to SetIoBuffer")
	}
	return fn(h)
}

func (d *Device) Start() (int32, error) {
	fn := d.StartImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to Start")
	}
	return fn()
}

func (d *Device) Stop() error {
	fn := d.StopImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to Stop")
	}
	return fn()
}

func (d *Device) ListenStart() (int32, error) {
	fn := d.ListenStartImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to ListenStart")
	}
	return fn()
}

func (d *Device) ListenStop() error {
	fn := d.ListenStopImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to ListenStop")
	}
	return fn()
}

func (d *Device) SetClientName(name string) (int32, error) {
	fn := d.SetClientNameImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to SetClientName")
	}
	return fn(name)
}

func (d *Device) GetStatus() (uint32, error) {
	fn := d.GetStatusImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to GetStatus")
	}
	return fn()
}

func (d *Device) SetPromiscuousMode(enabled bool) (int32, error) {
	fn := d.SetPromiscuousModeImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to SetPromiscuousMode")
	}
	return fn(enabled)
}

func (d *Device) ConfigMulticastAddMac(addr ethernet.MacAddress) (int32, error) {
	fn := d.ConfigMulticastAddMacImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to ConfigMulticastAddMac")
	}
	return fn(addr)
}

func (d *Device) ConfigMulticastDeleteMac(addr ethernet.MacAddress) (int32, error) {
	fn := d.ConfigMulticastDeleteMacImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to ConfigMulticastDeleteMac")
	}
	return fn(addr)
}

func (d *Device) ConfigMulticastSetPromiscuousMode(enabled bool) (int32, error) {
	fn := d.ConfigMulticastSetPromiscuousModeImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to ConfigMulticastSetPromiscuousMode")
	}
	return fn(enabled)
}

func (d *Device) ConfigMulticastTestFilter() (int32, error) {
	fn := d.ConfigMulticastTestFilterImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to ConfigMulticastTestFilter")
	}
	return fn()
}

func (d *Device) DumpRegisters() (int32, error) {
	fn := d.DumpRegistersImpl
	if fn == nil {
		d.Helper()
		d.Fatal("unexpected call to DumpRegisters")
	}
	return fn()
}
