package services

import (
	"context"
	"log"
	"time"

	"goodpack-server/utils"
)

// StartSnapshotScheduler fires an auto inventory snapshot at 23:59:59 on the last day of each month.
// Auto snapshots are skipped if a snapshot for that month already exists.
func StartSnapshotScheduler(service *InventorySnapshotService) {
	go func() {
		for {
			now := utils.NowInThailand()
			next := nextEndOfMonthAt235959(now)
			log.Printf("📅 Snapshot scheduler: next auto-snapshot at %s", next.Format("2006-01-02 15:04:05"))

			timer := time.NewTimer(next.Sub(now))
			<-timer.C

			t := utils.NowInThailand()
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

			log.Printf("📸 Running auto monthly snapshot for %02d/%d", int(t.Month()), t.Year())
			snap, err := service.TakeSnapshot(ctx, int(t.Month()), t.Year(), "system", false)
			if err != nil {
				log.Printf("❌ Auto snapshot failed: %v", err)
			} else {
				log.Printf("✅ Auto snapshot done: %d products, month=%d year=%d", snap.TotalProducts, snap.Month, snap.Year)
			}
			cancel()
		}
	}()
}

// nextEndOfMonthAt235959 returns 23:59:59 on the last day of the current month.
// If that moment has already passed today, it returns the same time next month.
func nextEndOfMonthAt235959(now time.Time) time.Time {
	loc := now.Location()

	// Last day of current month = first day of next month minus 1 second, then truncate to day
	firstOfNext := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, loc)
	lastDay := firstOfNext.Add(-24 * time.Hour)
	target := time.Date(lastDay.Year(), lastDay.Month(), lastDay.Day(), 23, 59, 59, 0, loc)

	if now.Before(target) {
		return target
	}

	// Current month's window passed; compute next month's last day
	firstOfNextNext := time.Date(now.Year(), now.Month()+2, 1, 0, 0, 0, 0, loc)
	lastDayNext := firstOfNextNext.Add(-24 * time.Hour)
	return time.Date(lastDayNext.Year(), lastDayNext.Month(), lastDayNext.Day(), 23, 59, 59, 0, loc)
}
