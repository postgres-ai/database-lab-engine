/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react';

import { Section } from '../components/Section';
import { Property } from '../components/Property';

export const Details = () => {
  return (
    <Section title='Details'>
      <Property name='Database docker image'>postgres:12</Property>
      <Property name='Version'>0.4.4</Property>
    </Section>
  );
};
