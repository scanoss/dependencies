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

// Package logger handles logging for everything in the dependency system
// It uses zap to achieve this
package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger
var S *zap.SugaredLogger

func NewDevLogger() error {
	var err error
	L, err = zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to load dev logger: %v", err)
	}
	return nil
}

func NewProdLogger() error {
	var err error
	L, err = zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to load prod logger: %v", err)
	}
	return nil
}

func NewProdLoggerLevel(lvl zapcore.Level) error {
	pc := zap.NewProductionConfig()
	pc.Level = zap.NewAtomicLevelAt(lvl)
	var err error
	L, err = pc.Build()
	if err != nil {
		return fmt.Errorf("failed to load prod logger: %v", err)
	}
	return nil
}

func NewSugaredDevLogger() error {
	if err := NewDevLogger(); err != nil {
		return err
	}
	S = L.Sugar()
	return nil
}

func NewSugaredProdLogger() error {
	if err := NewProdLogger(); err != nil {
		return err
	}
	S = L.Sugar()
	return nil
}

func NewSugaredProdLoggerLevel(lvl zapcore.Level) error {
	if err := NewProdLoggerLevel(lvl); err != nil {
		return err
	}
	S = L.Sugar()
	return nil
}

func SyncZap() {
	// Sync the Sugared logger if it's set
	if S != nil {
		err := S.Sync()
		if err != nil {
			fmt.Printf("Warning: Failed to sync zap: %v\n", err)
		}
	} else if L != nil { // Otherwise, sync the Logger if it's set
		err := L.Sync()
		if err != nil {
			fmt.Printf("Warning: Failed to sync zap: %v\n", err)
		}
	}
}
