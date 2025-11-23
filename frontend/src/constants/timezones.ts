/**
 * IANA Timezone Database
 * Comprehensive list of all commonly used timezones
 */

export const TIMEZONES = [
  // UTC/GMT
  { value: "UTC", title: "UTC (Coordinated Universal Time)" },
  { value: "GMT", title: "GMT (Greenwich Mean Time)" },

  // Americas - North America
  { value: "America/New_York", title: "America/New_York (Eastern Time)" },
  { value: "America/Chicago", title: "America/Chicago (Central Time)" },
  { value: "America/Denver", title: "America/Denver (Mountain Time)" },
  { value: "America/Phoenix", title: "America/Phoenix (Arizona, no DST)" },
  { value: "America/Los_Angeles", title: "America/Los_Angeles (Pacific Time)" },
  { value: "America/Anchorage", title: "America/Anchorage (Alaska)" },
  { value: "Pacific/Honolulu", title: "Pacific/Honolulu (Hawaii)" },
  { value: "America/Toronto", title: "America/Toronto (Toronto)" },
  { value: "America/Vancouver", title: "America/Vancouver (Vancouver)" },
  { value: "America/Halifax", title: "America/Halifax (Atlantic Time)" },
  { value: "America/St_Johns", title: "America/St_Johns (Newfoundland)" },

  // Americas - Latin America
  { value: "America/Mexico_City", title: "America/Mexico_City (Mexico City)" },
  { value: "America/Guatemala", title: "America/Guatemala (Guatemala)" },
  { value: "America/Belize", title: "America/Belize (Belize)" },
  { value: "America/El_Salvador", title: "America/El_Salvador (El Salvador)" },
  { value: "America/Managua", title: "America/Managua (Managua)" },
  { value: "America/Tegucigalpa", title: "America/Tegucigalpa (Tegucigalpa)" },
  { value: "America/Costa_Rica", title: "America/Costa_Rica (Costa Rica)" },
  { value: "America/Panama", title: "America/Panama (Panama)" },
  { value: "America/Havana", title: "America/Havana (Havana)" },
  { value: "America/Bogota", title: "America/Bogota (Bogota, Colombia)" },
  { value: "America/Lima", title: "America/Lima (Lima, Peru)" },
  { value: "America/Santiago", title: "America/Santiago (Santiago, Chile)" },
  { value: "America/Caracas", title: "America/Caracas (Caracas, Venezuela)" },
  { value: "America/La_Paz", title: "America/La_Paz (La Paz, Bolivia)" },
  { value: "America/Sao_Paulo", title: "America/Sao_Paulo (São Paulo, Brazil)" },
  { value: "America/Buenos_Aires", title: "America/Buenos_Aires (Buenos Aires)" },
  { value: "America/Montevideo", title: "America/Montevideo (Montevideo)" },
  { value: "America/Asuncion", title: "America/Asuncion (Asunción, Paraguay)" },

  // Europe - Western
  { value: "Europe/London", title: "Europe/London (London, UK)" },
  { value: "Europe/Dublin", title: "Europe/Dublin (Dublin, Ireland)" },
  { value: "Europe/Lisbon", title: "Europe/Lisbon (Lisbon, Portugal)" },
  { value: "Atlantic/Reykjavik", title: "Atlantic/Reykjavik (Reykjavik, Iceland)" },
  { value: "Atlantic/Azores", title: "Atlantic/Azores (Azores)" },
  { value: "Atlantic/Canary", title: "Atlantic/Canary (Canary Islands)" },

  // Europe - Central
  { value: "Europe/Paris", title: "Europe/Paris (Paris, France)" },
  { value: "Europe/Brussels", title: "Europe/Brussels (Brussels, Belgium)" },
  { value: "Europe/Madrid", title: "Europe/Madrid (Madrid, Spain)" },
  { value: "Europe/Berlin", title: "Europe/Berlin (Berlin, Germany)" },
  { value: "Europe/Rome", title: "Europe/Rome (Rome, Italy)" },
  { value: "Europe/Vienna", title: "Europe/Vienna (Vienna, Austria)" },
  { value: "Europe/Amsterdam", title: "Europe/Amsterdam (Amsterdam, Netherlands)" },
  { value: "Europe/Stockholm", title: "Europe/Stockholm (Stockholm, Sweden)" },
  { value: "Europe/Oslo", title: "Europe/Oslo (Oslo, Norway)" },
  { value: "Europe/Copenhagen", title: "Europe/Copenhagen (Copenhagen, Denmark)" },
  { value: "Europe/Zurich", title: "Europe/Zurich (Zurich, Switzerland)" },
  { value: "Europe/Prague", title: "Europe/Prague (Prague, Czech Republic)" },
  { value: "Europe/Warsaw", title: "Europe/Warsaw (Warsaw, Poland)" },
  { value: "Europe/Budapest", title: "Europe/Budapest (Budapest, Hungary)" },
  { value: "Europe/Belgrade", title: "Europe/Belgrade (Belgrade, Serbia)" },

  // Europe - Eastern
  { value: "Europe/Athens", title: "Europe/Athens (Athens, Greece)" },
  { value: "Europe/Bucharest", title: "Europe/Bucharest (Bucharest, Romania)" },
  { value: "Europe/Helsinki", title: "Europe/Helsinki (Helsinki, Finland)" },
  { value: "Europe/Kiev", title: "Europe/Kiev (Kiev, Ukraine)" },
  { value: "Europe/Sofia", title: "Europe/Sofia (Sofia, Bulgaria)" },
  { value: "Europe/Istanbul", title: "Europe/Istanbul (Istanbul, Turkey)" },
  { value: "Europe/Moscow", title: "Europe/Moscow (Moscow, Russia)" },
  { value: "Europe/Minsk", title: "Europe/Minsk (Minsk, Belarus)" },
  { value: "Europe/Tallinn", title: "Europe/Tallinn (Tallinn, Estonia)" },
  { value: "Europe/Riga", title: "Europe/Riga (Riga, Latvia)" },
  { value: "Europe/Vilnius", title: "Europe/Vilnius (Vilnius, Lithuania)" },

  // Middle East
  { value: "Asia/Dubai", title: "Asia/Dubai (Dubai, Abu Dhabi)" },
  { value: "Asia/Jerusalem", title: "Asia/Jerusalem (Jerusalem, Israel)" },
  { value: "Asia/Riyadh", title: "Asia/Riyadh (Riyadh, Saudi Arabia)" },
  { value: "Asia/Tehran", title: "Asia/Tehran (Tehran, Iran)" },
  { value: "Asia/Baghdad", title: "Asia/Baghdad (Baghdad, Iraq)" },
  { value: "Asia/Kuwait", title: "Asia/Kuwait (Kuwait)" },
  { value: "Asia/Bahrain", title: "Asia/Bahrain (Bahrain)" },
  { value: "Asia/Qatar", title: "Asia/Qatar (Qatar)" },
  { value: "Asia/Muscat", title: "Asia/Muscat (Muscat, Oman)" },
  { value: "Asia/Amman", title: "Asia/Amman (Amman, Jordan)" },
  { value: "Asia/Beirut", title: "Asia/Beirut (Beirut, Lebanon)" },
  { value: "Asia/Damascus", title: "Asia/Damascus (Damascus, Syria)" },

  // Africa
  { value: "Africa/Cairo", title: "Africa/Cairo (Cairo, Egypt)" },
  { value: "Africa/Johannesburg", title: "Africa/Johannesburg (Johannesburg, South Africa)" },
  { value: "Africa/Nairobi", title: "Africa/Nairobi (Nairobi, Kenya)" },
  { value: "Africa/Lagos", title: "Africa/Lagos (Lagos, Nigeria)" },
  { value: "Africa/Casablanca", title: "Africa/Casablanca (Casablanca, Morocco)" },
  { value: "Africa/Algiers", title: "Africa/Algiers (Algiers, Algeria)" },
  { value: "Africa/Tunis", title: "Africa/Tunis (Tunis, Tunisia)" },
  { value: "Africa/Accra", title: "Africa/Accra (Accra, Ghana)" },
  { value: "Africa/Addis_Ababa", title: "Africa/Addis_Ababa (Addis Ababa, Ethiopia)" },
  { value: "Africa/Dar_es_Salaam", title: "Africa/Dar_es_Salaam (Dar es Salaam, Tanzania)" },

  // Asia - Central
  { value: "Asia/Almaty", title: "Asia/Almaty (Almaty, Kazakhstan)" },
  { value: "Asia/Tashkent", title: "Asia/Tashkent (Tashkent, Uzbekistan)" },
  { value: "Asia/Yekaterinburg", title: "Asia/Yekaterinburg (Yekaterinburg, Russia)" },
  { value: "Asia/Omsk", title: "Asia/Omsk (Omsk, Russia)" },
  { value: "Asia/Novosibirsk", title: "Asia/Novosibirsk (Novosibirsk, Russia)" },
  { value: "Asia/Krasnoyarsk", title: "Asia/Krasnoyarsk (Krasnoyarsk, Russia)" },
  { value: "Asia/Irkutsk", title: "Asia/Irkutsk (Irkutsk, Russia)" },
  { value: "Asia/Yakutsk", title: "Asia/Yakutsk (Yakutsk, Russia)" },
  { value: "Asia/Vladivostok", title: "Asia/Vladivostok (Vladivostok, Russia)" },

  // Asia - South
  { value: "Asia/Kolkata", title: "Asia/Kolkata (India Standard Time)" },
  { value: "Asia/Karachi", title: "Asia/Karachi (Karachi, Pakistan)" },
  { value: "Asia/Dhaka", title: "Asia/Dhaka (Dhaka, Bangladesh)" },
  { value: "Asia/Colombo", title: "Asia/Colombo (Colombo, Sri Lanka)" },
  { value: "Asia/Kathmandu", title: "Asia/Kathmandu (Kathmandu, Nepal)" },
  { value: "Asia/Thimphu", title: "Asia/Thimphu (Thimphu, Bhutan)" },

  // Asia - Southeast
  { value: "Asia/Bangkok", title: "Asia/Bangkok (Bangkok, Thailand)" },
  { value: "Asia/Jakarta", title: "Asia/Jakarta (Jakarta, Indonesia)" },
  { value: "Asia/Singapore", title: "Asia/Singapore (Singapore)" },
  { value: "Asia/Kuala_Lumpur", title: "Asia/Kuala_Lumpur (Kuala Lumpur, Malaysia)" },
  { value: "Asia/Manila", title: "Asia/Manila (Manila, Philippines)" },
  { value: "Asia/Ho_Chi_Minh", title: "Asia/Ho_Chi_Minh (Ho Chi Minh, Vietnam)" },
  { value: "Asia/Phnom_Penh", title: "Asia/Phnom_Penh (Phnom Penh, Cambodia)" },
  { value: "Asia/Vientiane", title: "Asia/Vientiane (Vientiane, Laos)" },
  { value: "Asia/Yangon", title: "Asia/Yangon (Yangon, Myanmar)" },

  // Asia - East
  { value: "Asia/Hong_Kong", title: "Asia/Hong_Kong (Hong Kong)" },
  { value: "Asia/Shanghai", title: "Asia/Shanghai (Beijing, Shanghai, China)" },
  { value: "Asia/Taipei", title: "Asia/Taipei (Taipei, Taiwan)" },
  { value: "Asia/Tokyo", title: "Asia/Tokyo (Tokyo, Japan)" },
  { value: "Asia/Seoul", title: "Asia/Seoul (Seoul, South Korea)" },
  { value: "Asia/Pyongyang", title: "Asia/Pyongyang (Pyongyang, North Korea)" },
  { value: "Asia/Ulaanbaatar", title: "Asia/Ulaanbaatar (Ulaanbaatar, Mongolia)" },

  // Australia & Pacific
  { value: "Australia/Perth", title: "Australia/Perth (Perth, WA)" },
  { value: "Australia/Eucla", title: "Australia/Eucla (Eucla, WA)" },
  { value: "Australia/Adelaide", title: "Australia/Adelaide (Adelaide, SA)" },
  { value: "Australia/Darwin", title: "Australia/Darwin (Darwin, NT)" },
  { value: "Australia/Brisbane", title: "Australia/Brisbane (Brisbane, QLD)" },
  { value: "Australia/Sydney", title: "Australia/Sydney (Sydney, Melbourne, NSW)" },
  { value: "Australia/Melbourne", title: "Australia/Melbourne (Melbourne, VIC)" },
  { value: "Australia/Hobart", title: "Australia/Hobart (Hobart, TAS)" },
  { value: "Australia/Lord_Howe", title: "Australia/Lord_Howe (Lord Howe Island)" },
  { value: "Pacific/Auckland", title: "Pacific/Auckland (Auckland, New Zealand)" },
  { value: "Pacific/Fiji", title: "Pacific/Fiji (Fiji)" },
  { value: "Pacific/Guam", title: "Pacific/Guam (Guam)" },
  { value: "Pacific/Tahiti", title: "Pacific/Tahiti (Tahiti)" },
  { value: "Pacific/Tongatapu", title: "Pacific/Tongatapu (Tonga)" },
  { value: "Pacific/Port_Moresby", title: "Pacific/Port_Moresby (Port Moresby, PNG)" },
  { value: "Pacific/Samoa", title: "Pacific/Samoa (Samoa)" },
  { value: "Pacific/Chatham", title: "Pacific/Chatham (Chatham Islands, NZ)" },

  // Atlantic & Others
  { value: "Atlantic/Bermuda", title: "Atlantic/Bermuda (Bermuda)" },
  { value: "Atlantic/Cape_Verde", title: "Atlantic/Cape_Verde (Cape Verde)" },
  { value: "Atlantic/South_Georgia", title: "Atlantic/South_Georgia (South Georgia)" },
  { value: "Indian/Maldives", title: "Indian/Maldives (Maldives)" },
  { value: "Indian/Mauritius", title: "Indian/Mauritius (Mauritius)" },
  { value: "Indian/Reunion", title: "Indian/Reunion (Réunion)" },
];

/**
 * Get the browser's local timezone
 * @returns IANA timezone identifier (e.g., "America/New_York")
 */
export function getBrowserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}

/**
 * Get timezones list with browser timezone guaranteed to be included
 * @returns Array of timezone objects
 */
export function getTimezonesWithBrowser(): Array<{ value: string; title: string }> {
  const browserTz = getBrowserTimezone();
  
  // Check if browser timezone is already in the list
  const exists = TIMEZONES.some(tz => tz.value === browserTz);
  
  if (exists || browserTz === "UTC") {
    return TIMEZONES;
  }
  
  // Add browser timezone at the top of the list
  return [
    { value: browserTz, title: `${browserTz} (Browser Detected)` },
    ...TIMEZONES
  ];
}

/**
 * Check if a timezone string is valid
 * @param timezone - IANA timezone identifier
 * @returns true if timezone is valid
 */
export function isValidTimezone(timezone: string): boolean {
  try {
    Intl.DateTimeFormat(undefined, { timeZone: timezone });
    return true;
  } catch {
    return false;
  }
}

/**
 * Format timezone for display with current offset
 * @param timezone - IANA timezone identifier
 * @returns Formatted string like "America/New_York (UTC-5)"
 */
export function formatTimezoneWithOffset(timezone: string): string {
  try {
    const now = new Date();
    const formatter = new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      timeZoneName: "short",
    });
    
    const parts = formatter.formatToParts(now);
    const tzPart = parts.find((part) => part.type === "timeZoneName");
    const offset = tzPart?.value || "";
    
    return `${timezone} (${offset})`;
  } catch {
    return timezone;
  }
}

