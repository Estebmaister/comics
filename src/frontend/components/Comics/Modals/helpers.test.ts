import type { ChangeEvent } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { handleInputChange } from './helpers';

describe('handleInputChange', () => {
  it('stores numeric fields as numbers', () => {
    const setFormState = vi.fn();
    const onChange = handleInputChange(setFormState);

    onChange({
      target: {
        name: 'viewed_chap',
        value: '7',
        type: 'number',
      },
    } as ChangeEvent<HTMLInputElement>);

    expect(setFormState).toHaveBeenCalled();
    const updater = setFormState.mock.calls[0][0];
    expect(updater({ viewed_chap: 0 })).toEqual({ viewed_chap: 7 });
  });
});
