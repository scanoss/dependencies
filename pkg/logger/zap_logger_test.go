// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package logger

import (
	"go.uber.org/zap"
	"testing"
)

func TestZapDevSugar(t *testing.T) {
	err := NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer SyncZap()
	S.Debug("Debug test statement.")
}

func TestZapProdSugar(t *testing.T) {
	err := NewSugaredProdLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer SyncZap()
	S.Info("Info test statement.")
}

func TestZapProdSugarLevel(t *testing.T) {
	err := NewSugaredProdLoggerLevel(zap.DebugLevel)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer SyncZap()
	S.Info("Info test statement.")
}

func TestZapPro(t *testing.T) {
	S = nil
	err := NewProdLoggerLevel(zap.DebugLevel)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer SyncZap()
	L.Info("Info test statement.")
}
