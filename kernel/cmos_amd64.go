// SPDX-License-Identifier: Unlicense OR MIT

package kernel

// Functions for reading the real time clock of the CMOS.

// readCMOSTime reads the CMOS clock and converts it to UNIX time in
// seconds.
//go:nosplit
func readCMOSTime() int64 {
	waitForCMOS()
	t := readCMOSTime0()
	// The CMOS may be updating its time during our reading.
	// Read the time until it is stable.
	for {
		waitForCMOS()
		t2 := readCMOSTime0()
		if t2 == t {
			break
		}
		t = t2
	}
	return t
}

//go:nosplit
func readCMOSTime0() int64 {
	sec, min, hour := readCMOSReg(0x00), readCMOSReg(0x02), readCMOSReg(0x04)
	day, month, year, century := readCMOSReg(0x07), readCMOSReg(0x08), readCMOSReg(0x09), readCMOSReg(0x32)
	statusB := readCMOSReg(0x0b)
	pm := false
	// Check for 12-hour format.
	if statusB&1<<1 != 0 {
		// Read and reset PM bit.
		pm = hour&1<<7 != 0
		hour = hour & 0x7f
	}
	// Check for decimal format.
	if statusB&1<<2 == 0 {
		sec, min, hour = decToBin(sec), decToBin(min), decToBin(hour)
		day, month, year, century = decToBin(day), decToBin(month), decToBin(year), decToBin(century)
	}
	if pm {
		hour = (hour + 12) % 24
	}
	return cmosToUnix(int(century)*100+int(year), int(month)-1, int(day)-1, int(hour), int(min), int(sec))
}

// decToBin converts a decimal value to binary.
//go:nosplit
func decToBin(v uint8) uint8 {
	return (v & 0x0F) + ((v / 16) * 10)
}

// cmosToUnix is similar to time.Date(...).Unix(), except that we can't use
// that from the kernel. Note that the month and day are zero-based.
//go:nosplit
func cmosToUnix(year, month, day, hour, min, sec int) int64 {
	const (
		secondsPerMinute = 60
		secondsPerHour   = 60 * secondsPerMinute
		secondsPerDay    = 24 * secondsPerHour
		absoluteZeroYear = -292277022399
		internalYear     = 1

		absoluteToInternal int64 = (absoluteZeroYear - internalYear) * 365.2425 * secondsPerDay
		unixToInternal     int64 = (1969*365 + 1969/4 - 1969/100 + 1969/400) * secondsPerDay
		internalToUnix     int64 = -unixToInternal
		daysPer400Years          = 365*400 + 97
		daysPer100Years          = 365*100 + 24
		daysPer4Years            = 365*4 + 1
	)

	var daysBefore = [...]int32{
		0,
		31,
		31 + 28,
		31 + 28 + 31,
		31 + 28 + 31 + 30,
		31 + 28 + 31 + 30 + 31,
		31 + 28 + 31 + 30 + 31 + 30,
		31 + 28 + 31 + 30 + 31 + 30 + 31,
		31 + 28 + 31 + 30 + 31 + 30 + 31 + 31,
		31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30,
		31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31,
		31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30,
		31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30 + 31,
	}
	// Normalize.
	year, month = norm(year, month, 11)
	min, sec = norm(min, sec, 60)
	hour, min = norm(hour, min, 60)
	day, hour = norm(day, hour, 24)

	y := uint64(int64(year) - absoluteZeroYear)

	// Compute days since the absolute epoch.

	// Add in days from 400-year cycles.
	n := y / 400
	y -= 400 * n
	d := daysPer400Years * n

	// Add in 100-year cycles.
	n = y / 100
	y -= 100 * n
	d += daysPer100Years * n

	// Add in 4-year cycles.
	n = y / 4
	y -= 4 * n
	d += daysPer4Years * n

	// Add in non-leap years.
	n = y
	d += 365 * n

	// Add in days before this month.
	d += uint64(daysBefore[month])
	if isLeap(year) && month >= 2 /* March */ {
		d++ // February 29
	}

	// Add in days before today.
	d += uint64(day)

	// Add in time elapsed today.
	abs := d * secondsPerDay
	abs += uint64(hour*secondsPerHour + min*secondsPerMinute + sec)

	unix := int64(abs) + (absoluteToInternal + internalToUnix)

	return unix
}

//go:nosplit
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// norm returns nhi, nlo such that
//	hi * base + lo == nhi * base + nlo
//	0 <= nlo < base
//go:nosplit
func norm(hi, lo, base int) (nhi, nlo int) {
	if lo < 0 {
		n := (-lo-1)/base + 1
		hi -= n
		lo += n * base
	}
	if lo >= base {
		n := lo / base
		hi += n
		lo -= n * base
	}
	return hi, lo
}

// waitForCMOS waits for the CMOS busy flag to clear. The
// flag is in bit 7 of the status A register.
//go:nosplit
func waitForCMOS() {
	for {
		statusA := readCMOSReg(0x0a)
		if statusA&1<<7 == 0 {
			return
		}
	}
}

//go:nosplit
func readCMOSReg(reg uint8) uint8 {
	const (
		cmosAddr = 0x70
		cmosData = 0x71
	)
	outb(cmosAddr, reg)
	return inb(cmosData)
}
