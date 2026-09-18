export default class Boolset {
  private data: Uint8Array;
  constructor(data?: string, raw?: Uint8Array) {
    if (!data) {
      this.data = raw ? raw : new Uint8Array(0);
      return;
    }

    try {
      const binString = atob(data);
      var bytes = new Uint8Array(binString.length);
      for (var i = 0; i < binString.length; i++) {
        bytes[i] = binString.charCodeAt(i);
      }
      this.data = bytes;
    } catch (e) {
      console.warn("Failed to decode boolset", e);
      this.data = new Uint8Array(0);
    }
  }

  enabled(index: number): boolean {
    if (index >= this.data.length * 8) {
      return false;
    }
    return (this.data[Math.floor(index / 8)] & (1 << index % 8)) != 0;
  }

  and(other: Boolset): Boolset {
    const length = Math.max(this.data.length, other.data.length);
    const result = new Uint8Array(length);
    for (let i = 0; i < length; i++) {
      result[i] = (this.data[i] || 0) & (other.data[i] || 0);
    }
    return new Boolset(undefined, result);
  }

  or(other: Boolset): Boolset {
    const length = Math.max(this.data.length, other.data.length);
    const result = new Uint8Array(length);
    for (let i = 0; i < length; i++) {
      result[i] = (this.data[i] || 0) | (other.data[i] || 0);
    }
    return new Boolset(undefined, result);
  }

  set(index: number, enabled: boolean): Boolset {
    if (index >= this.data.length * 8) {
      const newData = new Uint8Array(Math.ceil((index + 1) / 8));
      newData.set(this.data);
      this.data = newData;
    }
    if (enabled) {
      this.data[Math.floor(index / 8)] |= 1 << index % 8;
    } else {
      this.data[Math.floor(index / 8)] &= ~(1 << index % 8);
    }
    return this;
  }

  sets(val: { [key: number]: boolean }) {
    for (const index in val) {
      this.set(parseInt(index), val[index]);
    }
  }

  /*

  func (b *BooleanSet) MarshalBinary() (data []byte, err error) {
	return *b, nil
}


  func (b *BooleanSet) String() (data string, err error) {
	raw, err := b.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(raw), nil
}

   */

  toString(): string {
    return btoa(String.fromCharCode.apply(null, Array.from(this.data)));
  }
}
