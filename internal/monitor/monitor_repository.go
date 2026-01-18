package monitor

import (
	"context"
	"dhruv/probe/internal/db"
	"fmt"
)

type Monitor struct {
}

func GetNextUrlsToPoll(ctx context.Context, db *db.DB) {

	rows, err := db.Pool.QueryContext(ctx, "select monitor_id, monitor_name from monitor")
	if err != nil {
		fmt.Printf("error querying %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var monitor_name string
		err := rows.Scan(&id, &monitor_name)
		if err != nil {
			continue
		}
		fmt.Printf("id: %d, monitor_name: %s\n", id, monitor_name)

	}
}
