package main

import "time"

func NewHPBar(maxHP int) *HPBar {
	return &HPBar{
		lastDamageTime: time.Time{}, // Initialize to zero time
		currentOpacity: 0.0,         // Start invisible
		maxHP:          maxHP,
		currentHP:      maxHP,
		visible:        false, // Start hidden
	}
}

func (bar *HPBar) updateOpacity() {
	timeSinceLastDamage := time.Since(bar.lastDamageTime)
	if timeSinceLastDamage > 5*time.Second {
		bar.visible = false
	}
}

func (bar *HPBar) AddHP(amount int) {
	bar.currentHP += amount
	bar.lastDamageTime = time.Now()
	bar.visible = true
	if bar.currentHP > bar.maxHP {
		bar.currentHP = bar.maxHP
	}
}
