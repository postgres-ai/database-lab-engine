/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import moment from 'moment';

const Format = {
  formatSeconds: function (seconds, decimal, separator = ' ') {
    let result = [];
    let hours = Math.floor(seconds / 3600);
    let minutes = Math.floor((seconds - (hours * 3600)) / 60);
    let secs = seconds - (hours * 3600) - (minutes * 60);

    if (hours) {
      result.push(hours + separator + 'h');
    }
    if (minutes) {
      result.push(minutes + separator + 'm');
    }
    if (secs) {
      result.push(secs.toFixed(decimal) + separator + 's');
    }

    return result.join(', ');
  },

  formatBytes: function (bytes, decimals, dec) {
    let k = 1024;
    let dm = decimals || 2;
    let sizes = ['bytes', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB',
      'EiB', 'ZiB', 'YiB'
    ];

    if (isNaN(bytes)) {
      return '';
    }

    if (bytes === 0) {
      return '0 Bytes';
    }

    let i = Math.floor(Math.log(bytes) / Math.log(k));
    let value = parseFloat((bytes / Math.pow(k, i)).toFixed(dm));

    if (value < 10 && !dec) {
      value = value * k;
      i--;
    }

    return value + ' ' + sizes[i];
  },

  formatGiB: function (bytes, decimals) {
    const GiB = 1024 * 1024 * 1024;
    const dm = decimals || 3;

    if (isNaN(bytes)) {
      return '';
    }

    if (bytes === 0) {
      return '0 Bytes';
    }

    if (bytes < 10 * 1024 * 1024 * 1024) {
      return this.formatBytes(bytes, decimals, true);
    }

    let num = parseFloat(bytes / GiB).toFixed(dm);

    return num + ' GiB';
  },

  formatNumber: function (num, decimals) {
    let k = 1000;
    let dm = decimals || 2;
    let sizes = ['k', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y'];

    if (num === 0) {
      return '0';
    }

    if (k > num) {
      return num;
    }

    let i = Math.floor(Math.log(num) / Math.log(k));
    let value = parseFloat((num / Math.pow(k, i)).toFixed(dm));

    if (value < 10) {
      value = value * k;
      i--;
    }

    return Math.round(value) + ' ' + sizes[i];
  },

  formatStatus: function (status) {
    let statusText = status.split('_').join(' ');

    return statusText[0].toUpperCase() + statusText.substring(1);
  },

  formatTimestampUtc: function (timestamp) {
    if (!timestamp) {
      return null;
    }

    return moment(timestamp).utc().format('YYYY-MM-DD HH:mm:ss UTC');
  },

  formatTimestamp: function (timestamp) {
    if (!timestamp) {
      return null;
    }

    return moment(timestamp).format('YYYY-MM-DD HH:mm:ss');
  },

  formatDate: function (timestamp) {
    if (!timestamp) {
      return null;
    }

    return moment(timestamp).format('YYYY-MM-DD');
  },

  formatUnixDate: function (timestamp) {
    if (!timestamp) {
      return null;
    }
    return moment.unix(timestamp).format('YYYY-MM-DD');
  },

  formatSql: function (sql) {
    let query = sql;
    const keywords = ['ADD CONSTRAINT', 'ALTER COLUMN', 'ALTER TABLE', 'ALTER', 'BACKUP DATABASE',
      'BETWEEN', 'CASE', 'CHECK', 'COLUMN', 'CONSTRAINT', 'CREATE DATABASE', 'CREATE INDEX',
      'CREATE OR REPLACE VIEW', 'CREATE TABLE', 'CREATE PROCEDURE', 'CREATE UNIQUE INDEX',
      'CREATE VIEW', 'CREATE', 'DATABASE', 'DEFAULT', 'DELETE', 'DESC', 'DISTINCT', 'DROP COLUMN',
      'DROP CONSTRAINT', 'DROP DATABASE', 'DROP DEFAULT', 'DROP INDEX', 'DROP TABLE', 'DROP VIEW',
      'DROP', 'EXEC', 'EXISTS', 'FOREIGN KEY', 'FROM', 'FULL OUTER JOIN', 'GROUP BY', 'HAVING',
      'INDEX', 'INNER JOIN', 'INSERT INTO', 'INSERT INTO SELECT', 'IN', 'IS NULL', 'IS NOT NULL',
      'JOIN', 'LEFT JOIN', 'LIKE', 'LIMIT', 'ORDER BY', 'OUTER JOIN', 'PRIMARY KEY', 'PROCEDURE',
      'RIGHT JOIN', 'ROWNUM', 'SELECT', 'SELECT DISTINCT', 'SELECT INTO', 'SELECT TOP', 'OFFSET',
      'TABLE', 'TRUNCATE TABLE', 'UNION ALL', 'UNIQUE', 'UPDATE', 'VALUES', 'VIEW', 'WHERE',
      'UNION', 'WITH', 'NOT', 'TOP', 'OR', 'ALL', 'AND', 'ANY', 'ADD', 'AS', 'ASC', 'SET', 'ON'];

    for (let i = 0; i < keywords.length; i++) {
      let regex = new RegExp('\\b' + keywords[i] + '\\b', 'gi');

      query = query.replace(regex, '<b>' + keywords[i] + '</b>');
    }

    return query;
  },

  quoteStr: function (text) {
    if (text.indexOf(' ') !== -1) {
      return '"' + text + '"';
    }

    return text;
  },

  formatDbLabSessionCheck: function (check) {
    switch (check) {
    case 'session-duration-acceptable':
    case 'session_duration_acceptable':
      return 'Session duration is within allowed interval';
    case 'no-long-dangerous-locks':
    case 'no_long_dangerous_locks':
      return 'Dangerous locks are not observed during the session';
    default:
      return check;
    }
  },

  limitStr: function (str, limit) {
    return (str.length > limit) ? str.substr(0, limit).concat('…') : str;
  }
};

export default Format;
