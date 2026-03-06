// Package leaderelection provides a simple leader election mechanism based on
// PostgreSQL advisory locks. It allows multiple instances of an application to
// coordinate and ensure that only one instance is the leader at any given time.
// Non-leader instances can poll for leadership status and take over leadership
// if the current leader fails or releases the lock. The leader has a health
// check in case the database connection is lost along with the lock.
package leaderelection
