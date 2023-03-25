#! /usr/bin/env bash
set -Eeo pipefail
set +x

if [ -z "${STORX_REDIS_HOST}" ]; then
	echo STORX_REDIS_HOST env var is required
	exit 1
fi

if [ -z "${STORX_REDIS_PORT}" ]; then
	echo STORX_REDIS_PORT env var is required
	exit 1
fi

if [ -z "${STORX_REDIS_DIR}" ]; then
	echo STORX_REDIS_DIR env var is required
	exit 1
fi

start() {
	if [ -f "${STORX_REDIS_DIR}/redis-server.pid" ]; then
		return
	fi

	if [ ! -f "${STORX_REDIS_DIR}/redis.conf" ]; then
		cat >>"${STORX_REDIS_DIR}/redis.conf" <<EOF
bind ${STORX_REDIS_HOST}
port ${STORX_REDIS_PORT}
timeout 0
databases 2
dbfilename redis.db
dir ${STORX_REDIS_DIR}
daemonize yes
loglevel warning
logfile ${STORX_REDIS_DIR}/redis-server.log
pidfile ${STORX_REDIS_DIR}/redis-server.pid
EOF
	fi

	redis-server "${STORX_REDIS_DIR}/redis.conf"
}

stop() {
	# if the file exists, then Redis should be running
	if [ -f "${STORX_REDIS_DIR}/redis-server.pid" ]; then
		if ! redis-cli -h "${STORX_REDIS_HOST}" -p "${STORX_REDIS_PORT}" shutdown; then
			echo "******************************** REDIS SERVER LOG (last 25 lines) ********************************"
			echo "Printing the last 25 lines"
			echo
			tail -n 25 "${STORX_REDIS_DIR}/redis-server.log" || true
			echo
			echo "************************************** END REDIS SERVER LOG **************************************"
		fi
	fi
}

case "${1}" in
start) start ;;
stop) stop ;;
*) echo "the script must be executed as: $(basename "${BASH_SOURCE[0]}") start | stop" ;;
esac
