export const validateCron = (value: string) => {
    if (!value) return true;

    const parts = value.split(' ');
    if (parts.length !== 5) return false;

    const patterns = [
        /^(\*|[0-5]?\d|\*\/[1-9][0-9]*)$/, // minute (0-59)
        /^(\*|1?[0-9]|2[0-3]|\*\/[1-9][0-9]*)$/, // hour (0-23)
        /^(\*|[1-9]|[12][0-9]|3[01]|\*\/[1-9][0-9]*|\?)$/, // day (1-31)
        /^(\*|[1-9]|1[0-2]|\*\/[1-9][0-9]*)$/, // month (1-12)
        /^(\*|[0-6]|\*\/[1-9][0-9]*|\?)$/, // week (0-6)
    ];

    return parts.every((part, index) => {
        const pattern = patterns[index];
        return pattern ? pattern.test(part) : false;
    });
}; 