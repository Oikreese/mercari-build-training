import { useRef, useState } from 'react';
import { postItem } from '~/api';

interface Prop {
  onListingCompleted: () => void;
}

type FormDataType = {
  name: string;
  category: string;
  image: string | File;
};

export const Listing = ({ onListingCompleted }: Prop) => {
  const initialState = {
    name: '',
    category: '',
    image: '',
  };
  const [values, setValues] = useState<FormDataType>(initialState);

  const uploadImageRef = useRef<HTMLInputElement>(null);

  const onValueChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setValues({
      ...values,
      [event.target.name]: event.target.value,
    });
  };
  const onFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setValues({
      ...values,
      [event.target.name]: event.target.files![0],
    });
  };
  const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    // Validate field before submit
    const REQUIRED_FILEDS = ['name', 'image'];
    const missingFields = Object.entries(values)
      .filter(([, value]) => !value && REQUIRED_FILEDS.includes(value))
      .map(([key]) => key);

    if (missingFields.length) {
      alert(`Missing fields: ${missingFields.join(', ')}`);
      return;
    }

    // Submit the form
    postItem({
      name: values.name,
      category: values.category,
      image: values.image,
    })
      .then(() => {
        alert('Item listed successfully');
      })
      .catch((error) => {
        console.error('POST error:', error);
        alert('Failed to list this item');
      })
      .finally(() => {
        onListingCompleted();
        setValues(initialState);
        if (uploadImageRef.current) {
          uploadImageRef.current.value = '';
        }
      });
  };
  return (
    <div className="Listing">
      <form onSubmit={onSubmit} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '10px' }}>
        <input
          type="text"
          name="name"
          id="name"
          placeholder="name"
          onChange={onValueChange}
          required
          value={values.name}
          style={{ padding: '8px', borderRadius: '5px', border: '1px solid #a96e27' }}
        />
        <input
          type="text"
          name="category"
          id="category"
          placeholder="category"
          onChange={onValueChange}
          value={values.category}
          style={{ padding: '8px', borderRadius: '5px', border: '1px solid #a96e27' }}
        />
        <input
          type="file"
          name="image"
          id="image"
          onChange={onFileChange}
          required
          ref={uploadImageRef}
          style={{ padding: '8px', borderRadius: '5px', border: '1px solid #a96e27', backgroundColor: 'white' }}
        />
        <button
          type="submit"
          style={{
            padding: '10px 15px',
            backgroundColor: '#d7384a',
            color: 'white',
            border: 'none',
            borderRadius: '5px',
            cursor: 'pointer',
            fontWeight: 'bold',
            boxShadow: '4px 4px 10px rgba(0, 0, 0, 0.2)',
          }}
        >
          List this item
        </button>
      </form>
    </div>
  );
};
