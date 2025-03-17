import { useEffect, useState } from 'react';
import { Item, fetchItems, searchItems } from '~/api';

const PLACEHOLDER_IMAGE = import.meta.env.VITE_FRONTEND_URL + '/logo192.png';

interface Prop {
  reload: boolean;
  onLoadCompleted: () => void;
  keyword: string;
}

export const ItemList = ({ reload, onLoadCompleted, keyword }: Prop) => {
  const [items, setItems] = useState<Item[]>([]);
  const [searchItemResult, setSearchItemResult] = useState<Item[]>([]);
  useEffect(() => {
    const fetchData = () => {
      fetchItems()
        .then((data) => {
          console.debug('GET success:', data);
          setItems(data.items);
          onLoadCompleted();
        })
        .catch((error) => {
          console.error('GET error:', error);
        });
    };

    if (reload) {
      fetchData();
    }
  }, [reload, onLoadCompleted]);

  useEffect(() => {
    if (keyword) {
      searchItems({keyword})
        .then((data) => {
          console.debug('GET success:', data);
          const adaptedData = data.items.map((item) => ({
            ...item,
            category_name: item.category,
          }));
          setSearchItemResult(adaptedData);
        })
        .catch((error) => {
          console.error('GET error:', error);
        });
    }
  }, [keyword]);

  const displayItems = keyword ? searchItemResult : items;

  return (
    <div className="ItemListContainer">
      {displayItems?.map((item) => (
        <div key={item.id} className="ItemList">
          <img
            src={`http://localhost:9000/image/${item.image_name}`}
            alt="Item Image"
            onError={(e) => {
              e.currentTarget.src = PLACEHOLDER_IMAGE;
              e.currentTarget.alt = 'Image not found';
              e.currentTarget.title = 'Failed to load image';
            }}
          />
          <p>
            <span className="ItemName">Name: {item.name}</span>
            <br />
            <span className="ItemCategory">Category: {item.category_name}</span>
          </p>
        </div>
      ))}
    </div>
  );
};
