import { useState, useRef } from 'react';
import './App.css';
import { ItemList } from '~/components/ItemList';
import { Listing } from '~/components/Listing';
import { debounce } from 'lodash';

function App() {
  // reload ItemList after Listing complete
  const [reload, setReload] = useState(true);
  const [keyword, setKeyword] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceSetKeyword = debounce(setKeyword, 300);
  
  return (
    <div>
      <header className="Title">
        <p>
          <b>Simple Mercari</b>
        </p>
      </header>
      <div>
        <Listing onListingCompleted={() => setReload(true)} />
        <input className='filter'
          ref={inputRef} 
          type="text" 
          placeholder="search by name" 
          onChange={(e) => debounceSetKeyword(e.target.value)}
          style={{
            textAlign: 'center',
            margin: '20px auto',
            padding: '10px 15px',
            borderRadius: '10px',
            outline: 'none',
            width: '150px',
            boxShadow: '2px 2px 5px rgba(0, 0, 0, 0.1)',
            transition: 'all 0.3s ease',
            display: 'block',
          }}
        />
      </div>
      <div>
        <ItemList reload={reload} onLoadCompleted={() => setReload(false)} keyword={keyword} />
      </div>
    </div>
  );
}

export default App;
