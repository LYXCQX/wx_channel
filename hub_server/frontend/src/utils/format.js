export function formatTime(timestamp) {
    if (!timestamp) return '-'
    return new Date(timestamp).toLocaleString('zh-CN', {
        hour12: false,
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    })
}

export function timeAgo(timestamp) {
    if (!timestamp) return ''
    const seconds = Math.floor((new Date() - new Date(timestamp)) / 1000)

    let interval = seconds / 31536000
    if (interval > 1) return Math.floor(interval) + " 年前"

    interval = seconds / 2592000
    if (interval > 1) return Math.floor(interval) + " 个月前"

    interval = seconds / 86400
    if (interval > 1) return Math.floor(interval) + " 天前"

    interval = seconds / 3600
    if (interval > 1) return Math.floor(interval) + " 小时前"

    interval = seconds / 60
    if (interval > 1) return Math.floor(interval) + " 分钟前"

    return Math.floor(seconds) + " 秒前"
}

export function formatDuration(seconds) {
    if (!seconds) return '0:00'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = Math.floor(seconds % 60)

    if (h > 0) {
        return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
    }
    return `${m}:${s.toString().padStart(2, '0')}`
}
